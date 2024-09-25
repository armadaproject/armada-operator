/*
Copyright 2022.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package install

import (
	"context"
	"time"

	"github.com/pkg/errors"

	"k8s.io/utils/ptr"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	installv1alpha1 "github.com/armadaproject/armada-operator/api/install/v1alpha1"
	"github.com/armadaproject/armada-operator/internal/controller/builders"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/yaml"
)

// LookoutReconciler reconciles a Lookout object
type LookoutReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=install.armadaproject.io,resources=lookouts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=install.armadaproject.io,resources=lookouts/status,verbs=get;update;patch
//+kubebuilder:rbac:groups="batch",resources=jobs;cronjobs,verbs=get;list;watch;create;delete;deletecollection;patch;update
//+kubebuilder:rbac:groups=monitoring.coreos.com,resources=prometheusrules;servicemonitors,verbs=get;list;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *LookoutReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("namespace", req.Namespace, "name", req.Name)

	started := time.Now()

	logger.Info("Reconciling object")

	var lookout installv1alpha1.Lookout
	if miss, err := getObject(ctx, r.Client, &lookout, req.NamespacedName, logger); err != nil || miss {
		return ctrl.Result{}, err
	}

	finish, err := checkAndHandleObjectDeletion(ctx, r.Client, &lookout, operatorFinalizer, nil, logger)
	if err != nil || finish {
		return ctrl.Result{}, err
	}

	pc, err := installv1alpha1.BuildPortConfig(lookout.Spec.ApplicationConfig)
	if err != nil {
		return ctrl.Result{}, err
	}
	lookout.Spec.PortConfig = pc

	var components *CommonComponents
	components, err = generateLookoutInstallComponents(&lookout, r.Scheme)
	if err != nil {
		return ctrl.Result{}, err
	}

	componentsCopy := components.DeepCopy()

	mutateFn := func() error {
		components.ReconcileComponents(componentsCopy)
		return nil
	}

	if err := upsertObjectIfNeeded(ctx, r.Client, components.ServiceAccount, lookout.Kind, mutateFn, logger); err != nil {
		return ctrl.Result{}, err
	}

	if err := upsertObjectIfNeeded(ctx, r.Client, components.Secret, lookout.Kind, mutateFn, logger); err != nil {
		return ctrl.Result{}, err
	}

	for _, job := range components.Jobs {
		err = func(job *batchv1.Job) error {
			if err := upsertObjectIfNeeded(ctx, r.Client, job, lookout.Kind, mutateFn, logger); err != nil {
				return err
			}

			if err := waitForJob(ctx, r.Client, job, jobPollInterval, jobTimeout); err != nil {
				return err
			}

			return nil
		}(job)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	if err := upsertObjectIfNeeded(ctx, r.Client, components.Deployment, lookout.Kind, mutateFn, logger); err != nil {
		return ctrl.Result{}, err
	}

	if err := upsertObjectIfNeeded(ctx, r.Client, components.Service, lookout.Kind, mutateFn, logger); err != nil {
		return ctrl.Result{}, err
	}

	if err := upsertObjectIfNeeded(ctx, r.Client, components.IngressHttp, lookout.Kind, mutateFn, logger); err != nil {
		return ctrl.Result{}, err
	}

	if err := upsertObjectIfNeeded(ctx, r.Client, components.CronJob, lookout.Kind, mutateFn, logger); err != nil {
		return ctrl.Result{}, err
	}

	if err := upsertObjectIfNeeded(ctx, r.Client, components.ServiceMonitor, lookout.Kind, mutateFn, logger); err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("Successfully reconciled resource", "durationMillis", time.Since(started).Milliseconds())

	return ctrl.Result{}, nil
}

type LookoutConfig struct {
	Postgres PostgresConfig
}

func generateLookoutInstallComponents(lookout *installv1alpha1.Lookout, scheme *runtime.Scheme) (*CommonComponents, error) {
	secret, err := builders.CreateSecret(lookout.Spec.ApplicationConfig, lookout.Name, lookout.Namespace, GetConfigFilename(lookout.Name))
	if err != nil {
		return nil, err
	}
	if err := controllerutil.SetOwnerReference(lookout, secret, scheme); err != nil {
		return nil, err
	}

	var serviceAccount *corev1.ServiceAccount
	serviceAccountName := lookout.Spec.CustomServiceAccount
	if serviceAccountName == "" {
		serviceAccount = builders.CreateServiceAccount(lookout.Name, lookout.Namespace, AllLabels(lookout.Name, lookout.Labels), lookout.Spec.ServiceAccount)
		if err = controllerutil.SetOwnerReference(lookout, serviceAccount, scheme); err != nil {
			return nil, errors.WithStack(err)
		}
		serviceAccountName = serviceAccount.Name
	}

	deployment, err := createLookoutDeployment(lookout, serviceAccountName)
	if err != nil {
		return nil, err
	}
	if err := controllerutil.SetOwnerReference(lookout, deployment, scheme); err != nil {
		return nil, err
	}

	service := builders.Service(lookout.Name, lookout.Namespace, AllLabels(lookout.Name, lookout.Labels), IdentityLabel(lookout.Name), lookout.Spec.PortConfig)
	if err := controllerutil.SetOwnerReference(lookout, service, scheme); err != nil {
		return nil, err
	}

	var serviceMonitor *monitoringv1.ServiceMonitor
	if lookout.Spec.Prometheus != nil && lookout.Spec.Prometheus.Enabled {
		serviceMonitor = createLookoutServiceMonitor(lookout)
		if err := controllerutil.SetOwnerReference(lookout, serviceMonitor, scheme); err != nil {
			return nil, err
		}
	}

	job, err := createLookoutMigrationJob(lookout, serviceAccountName)
	if err != nil {
		return nil, err
	}
	if err := controllerutil.SetOwnerReference(lookout, job, scheme); err != nil {
		return nil, err
	}

	var cronJob *batchv1.CronJob
	if enabled := lookout.Spec.DbPruningEnabled; enabled != nil && *enabled {
		cronJob, err = createLookoutCronJob(lookout)
		if err != nil {
			return nil, err
		}
		if err := controllerutil.SetOwnerReference(lookout, cronJob, scheme); err != nil {
			return nil, err
		}
	}

	ingressHttp, err := createLookoutIngressHttp(lookout)
	if err != nil {
		return nil, err
	}
	if ingressHttp != nil {
		if err := controllerutil.SetOwnerReference(lookout, ingressHttp, scheme); err != nil {
			return nil, err
		}
	}

	return &CommonComponents{
		Deployment:     deployment,
		Service:        service,
		ServiceAccount: serviceAccount,
		Secret:         secret,
		IngressHttp:    ingressHttp,
		Jobs:           []*batchv1.Job{job},
		ServiceMonitor: serviceMonitor,
		CronJob:        cronJob,
	}, nil
}

// createLookoutServiceMonitor will return a ServiceMonitor for this
func createLookoutServiceMonitor(lookout *installv1alpha1.Lookout) *monitoringv1.ServiceMonitor {
	return &monitoringv1.ServiceMonitor{
		TypeMeta: metav1.TypeMeta{
			Kind: "ServiceMonitor",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      lookout.Name,
			Namespace: lookout.Namespace,
			Labels:    AllLabels(lookout.Name, lookout.Spec.Labels, lookout.Spec.Prometheus.Labels),
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			Endpoints: []monitoringv1.Endpoint{
				{Port: "metrics", Interval: "15s"},
			},
		},
	}
}

// Function to build the deployment object for Lookout.
// This should be changing from CRD to CRD.  Not sure if generailize this helps much
func createLookoutDeployment(lookout *installv1alpha1.Lookout, serviceAccountName string) (*appsv1.Deployment, error) {
	var runAsUser int64 = 1000
	var runAsGroup int64 = 2000
	allowPrivilegeEscalation := false
	env := createEnv(lookout.Spec.Environment)
	volumes := createVolumes(lookout.Name, lookout.Spec.AdditionalVolumes)
	volumeMounts := createVolumeMounts(GetConfigFilename(lookout.Name), lookout.Spec.AdditionalVolumeMounts)

	deployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: lookout.Name, Namespace: lookout.Namespace, Labels: AllLabels(lookout.Name, lookout.Labels)},
		Spec: appsv1.DeploymentSpec{
			Replicas: lookout.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: IdentityLabel(lookout.Name),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:        lookout.Name,
					Namespace:   lookout.Namespace,
					Labels:      AllLabels(lookout.Name, lookout.Labels),
					Annotations: map[string]string{"checksum/config": GenerateChecksumConfig(lookout.Spec.ApplicationConfig.Raw)},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName:            serviceAccountName,
					TerminationGracePeriodSeconds: lookout.DeletionGracePeriodSeconds,
					SecurityContext: &corev1.PodSecurityContext{
						RunAsUser:  &runAsUser,
						RunAsGroup: &runAsGroup,
					},
					Affinity: &corev1.Affinity{
						PodAffinity: &corev1.PodAffinity{
							PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{{
								Weight: 100,
								PodAffinityTerm: corev1.PodAffinityTerm{
									TopologyKey: "kubernetes.io/hostname",
									LabelSelector: &metav1.LabelSelector{
										MatchExpressions: []metav1.LabelSelectorRequirement{{
											Key:      "app",
											Operator: metav1.LabelSelectorOpIn,
											Values:   []string{lookout.Name},
										}},
									},
								},
							}},
						},
					},
					Containers: []corev1.Container{{
						Name:            "lookout",
						ImagePullPolicy: "IfNotPresent",
						Image:           ImageString(lookout.Spec.Image),
						Args:            []string{appConfigFlag, appConfigFilepath},
						Ports: []corev1.ContainerPort{
							{
								Name:          "metrics",
								ContainerPort: lookout.Spec.PortConfig.MetricsPort,
								Protocol:      "TCP",
							},
							{
								Name:          "http",
								ContainerPort: lookout.Spec.PortConfig.HttpPort,
								Protocol:      "TCP",
							},
							{
								Name:          "grpc",
								ContainerPort: lookout.Spec.PortConfig.GrpcPort,
								Protocol:      "TCP",
							},
						},
						Env:             env,
						VolumeMounts:    volumeMounts,
						SecurityContext: &corev1.SecurityContext{AllowPrivilegeEscalation: &allowPrivilegeEscalation},
					}},
					Volumes: volumes,
				},
			},
		},
	}
	if lookout.Spec.Resources != nil {
		deployment.Spec.Template.Spec.Containers[0].Resources = *lookout.Spec.Resources
		deployment.Spec.Template.Spec.Containers[0].Env = addGoMemLimit(deployment.Spec.Template.Spec.Containers[0].Env, *lookout.Spec.Resources)
	}

	return &deployment, nil
}

func createLookoutIngressHttp(lookout *installv1alpha1.Lookout) (*networking.Ingress, error) {
	if len(lookout.Spec.HostNames) == 0 {
		// when no hostnames, no ingress can be configured
		return nil, nil
	}
	ingressName := lookout.Name + "-rest"
	ingressHttp := &networking.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name: ingressName, Namespace: lookout.Namespace, Labels: AllLabels(lookout.Name, lookout.Labels),
			Annotations: map[string]string{
				"kubernetes.io/ingress.class":              lookout.Spec.Ingress.IngressClass,
				"nginx.ingress.kubernetes.io/ssl-redirect": "true",
			},
		},
	}

	if lookout.Spec.ClusterIssuer != "" {
		ingressHttp.ObjectMeta.Annotations["certmanager.k8s.io/cluster-issuer"] = lookout.Spec.ClusterIssuer
		ingressHttp.ObjectMeta.Annotations["cert-manager.io/cluster-issuer"] = lookout.Spec.ClusterIssuer
	}

	if lookout.Spec.Ingress.Annotations != nil {
		for key, value := range lookout.Spec.Ingress.Annotations {
			ingressHttp.ObjectMeta.Annotations[key] = value
		}
	}
	ingressHttp.ObjectMeta.Labels = AllLabels(lookout.Name, lookout.Spec.Labels, lookout.Spec.Ingress.Labels)

	secretName := lookout.Name + "-service-tls"
	ingressHttp.Spec.TLS = []networking.IngressTLS{{Hosts: lookout.Spec.HostNames, SecretName: secretName}}
	var ingressRules []networking.IngressRule
	serviceName := lookout.Name
	for _, val := range lookout.Spec.HostNames {
		ingressRules = append(ingressRules, networking.IngressRule{Host: val, IngressRuleValue: networking.IngressRuleValue{
			HTTP: &networking.HTTPIngressRuleValue{
				Paths: []networking.HTTPIngressPath{{
					Path:     "/",
					PathType: (*networking.PathType)(ptr.To[string]("Prefix")),
					Backend: networking.IngressBackend{
						Service: &networking.IngressServiceBackend{
							Name: serviceName,
							Port: networking.ServiceBackendPort{
								Number: lookout.Spec.PortConfig.HttpPort,
							},
						},
					},
				}},
			},
		}})
	}
	ingressHttp.Spec.Rules = ingressRules

	return ingressHttp, nil
}

// createLookoutMigrationJob returns a batch Job or an error if the app config is not correct
func createLookoutMigrationJob(lookout *installv1alpha1.Lookout, serviceAccountName string) (*batchv1.Job, error) {
	runAsUser := int64(1000)
	runAsGroup := int64(2000)
	var terminationGracePeriodSeconds int64
	if lookout.Spec.TerminationGracePeriodSeconds != nil {
		terminationGracePeriodSeconds = *lookout.Spec.TerminationGracePeriodSeconds
	}
	allowPrivilegeEscalation := false
	parallelism := int32(1)
	completions := int32(1)
	backoffLimit := int32(0)
	env := lookout.Spec.Environment
	volumes := createVolumes(lookout.Name, lookout.Spec.AdditionalVolumes)
	volumeMounts := createVolumeMounts(GetConfigFilename(lookout.Name), lookout.Spec.AdditionalVolumeMounts)

	appConfig, err := builders.ConvertRawExtensionToYaml(lookout.Spec.ApplicationConfig)
	if err != nil {
		return nil, err
	}
	var lookoutConfig LookoutConfig
	err = yaml.Unmarshal([]byte(appConfig), &lookoutConfig)
	if err != nil {
		return nil, err
	}

	job := batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:        lookout.Name + "-migration",
			Namespace:   lookout.Namespace,
			Labels:      AllLabels(lookout.Name, lookout.Labels),
			Annotations: map[string]string{"checksum/config": GenerateChecksumConfig(lookout.Spec.ApplicationConfig.Raw)},
		},
		Spec: batchv1.JobSpec{
			Parallelism:  &parallelism,
			Completions:  &completions,
			BackoffLimit: &backoffLimit,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      lookout.Name + "-migration",
					Namespace: lookout.Namespace,
					Labels:    AllLabels(lookout.Name, lookout.Labels),
				},
				Spec: corev1.PodSpec{
					ServiceAccountName:            serviceAccountName,
					RestartPolicy:                 "Never",
					TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
					SecurityContext: &corev1.PodSecurityContext{
						RunAsUser:  &runAsUser,
						RunAsGroup: &runAsGroup,
					},
					InitContainers: []corev1.Container{{
						Name:  "lookout-migration-db-wait",
						Image: "postgres:15.2-alpine",
						Command: []string{
							"/bin/sh",
							"-c",
							`echo "Waiting for Postres..."
                                                         while ! nc -z $PGHOST $PGPORT; do
                                                           sleep 1
                                                         done
                                                         echo "Postres started!"
							 echo "Creating DB $PGDB if needed..."
							 psql -v ON_ERROR_STOP=1 --username "$PGUSER" -c "CREATE DATABASE $PGDB"
							 psql -v ON_ERROR_STOP=1 --username "$PGUSER" -c "GRANT ALL PRIVILEGES ON DATABASE $PGDB TO $PGUSER"
							 echo "DB $PGDB created"`,
						},
						Env: []corev1.EnvVar{
							{
								Name:  "PGHOST",
								Value: lookoutConfig.Postgres.Connection.Host,
							},
							{
								Name:  "PGPORT",
								Value: lookoutConfig.Postgres.Connection.Port,
							},
							{
								Name:  "PGUSER",
								Value: lookoutConfig.Postgres.Connection.User,
							},
							{
								Name:  "PGPASSWORD",
								Value: lookoutConfig.Postgres.Connection.Password,
							},
							{
								Name:  "PGDB",
								Value: lookoutConfig.Postgres.Connection.Dbname,
							},
						},
					}},
					Containers: []corev1.Container{{
						Name:            "lookout-migration",
						ImagePullPolicy: "IfNotPresent",
						Image:           ImageString(lookout.Spec.Image),
						Args: []string{
							"--migrateDatabase",
							appConfigFlag,
							appConfigFilepath,
						},
						Ports: []corev1.ContainerPort{{
							Name:          "metrics",
							ContainerPort: lookout.Spec.PortConfig.MetricsPort,
							Protocol:      "TCP",
						}},
						Env:             env,
						VolumeMounts:    volumeMounts,
						SecurityContext: &corev1.SecurityContext{AllowPrivilegeEscalation: &allowPrivilegeEscalation},
					}},
					NodeSelector: lookout.Spec.NodeSelector,
					Tolerations:  lookout.Spec.Tolerations,
					Volumes:      volumes,
				},
			},
		},
	}

	return &job, nil
}

// createLookoutCronJob returns a batch CronJob or an error if the app config is not correct
func createLookoutCronJob(lookout *installv1alpha1.Lookout) (*batchv1.CronJob, error) {
	runAsUser := int64(1000)
	runAsGroup := int64(2000)
	terminationGracePeriodSeconds := int64(0)
	if lookout.Spec.TerminationGracePeriodSeconds != nil {
		terminationGracePeriodSeconds = *lookout.Spec.TerminationGracePeriodSeconds
	}
	allowPrivilegeEscalation := false
	parallelism := int32(1)
	completions := int32(1)
	backoffLimit := int32(0)
	env := lookout.Spec.Environment
	volumes := createVolumes(lookout.Name, lookout.Spec.AdditionalVolumes)
	volumeMounts := createVolumeMounts(GetConfigFilename(lookout.Name), lookout.Spec.AdditionalVolumeMounts)
	dbPruningSchedule := "@hourly"
	if lookout.Spec.DbPruningSchedule != nil {
		dbPruningSchedule = *lookout.Spec.DbPruningSchedule
	}

	appConfig, err := builders.ConvertRawExtensionToYaml(lookout.Spec.ApplicationConfig)
	if err != nil {
		return nil, err
	}
	var lookoutConfig LookoutConfig
	err = yaml.Unmarshal([]byte(appConfig), &lookoutConfig)
	if err != nil {
		return nil, err
	}

	job := batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:        lookout.Name + "-db-pruner",
			Namespace:   lookout.Namespace,
			Labels:      AllLabels(lookout.Name, lookout.Labels),
			Annotations: map[string]string{"checksum/config": GenerateChecksumConfig(lookout.Spec.ApplicationConfig.Raw)},
		},
		Spec: batchv1.CronJobSpec{
			Schedule: dbPruningSchedule,
			JobTemplate: batchv1.JobTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      lookout.Name + "-db-pruner",
					Namespace: lookout.Namespace,
					Labels:    AllLabels(lookout.Name, lookout.Labels),
				},
				Spec: batchv1.JobSpec{
					Parallelism:  &parallelism,
					Completions:  &completions,
					BackoffLimit: &backoffLimit,
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Name:      lookout.Name + "-db-pruner",
							Namespace: lookout.Namespace,
							Labels:    AllLabels(lookout.Name, lookout.Labels),
						},
						Spec: corev1.PodSpec{
							RestartPolicy:                 "Never",
							TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
							SecurityContext: &corev1.PodSecurityContext{
								RunAsUser:  &runAsUser,
								RunAsGroup: &runAsGroup,
							},
							InitContainers: []corev1.Container{{
								Name:  "lookout-db-pruner-db-wait",
								Image: "alpine:3.10",
								Command: []string{
									"/bin/sh",
									"-c",
									`echo "Waiting for Postres..."
                                                         while ! nc -z $PGHOST $PGPORT; do
                                                           sleep 1
                                                         done
                                                         echo "Postres started!"`,
								},
								Env: []corev1.EnvVar{
									{
										Name:  "PGHOST",
										Value: lookoutConfig.Postgres.Connection.Host,
									},
									{
										Name:  "PGPORT",
										Value: lookoutConfig.Postgres.Connection.Port,
									},
								},
							}},
							Containers: []corev1.Container{{
								Name:            "lookout-db-pruner",
								ImagePullPolicy: "IfNotPresent",
								Image:           ImageString(lookout.Spec.Image),
								Args: []string{
									"--pruneDatabase",
									appConfigFlag,
									appConfigFilepath,
								},
								Ports: []corev1.ContainerPort{{
									Name:          "metrics",
									ContainerPort: lookout.Spec.PortConfig.MetricsPort,
									Protocol:      "TCP",
								}},
								Env:             env,
								VolumeMounts:    volumeMounts,
								SecurityContext: &corev1.SecurityContext{AllowPrivilegeEscalation: &allowPrivilegeEscalation},
							}},
							NodeSelector: lookout.Spec.NodeSelector,
							Tolerations:  lookout.Spec.Tolerations,
							Volumes:      volumes,
						},
					},
				},
			},
		},
	}

	return &job, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *LookoutReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&installv1alpha1.Lookout{}).
		Complete(r)
}
