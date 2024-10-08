/*
Copyright 2023.
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
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/apimachinery/pkg/util/intstr"

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

// SchedulerReconciler reconciles a Scheduler object
type SchedulerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=install.armadaproject.io,resources=schedulers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=install.armadaproject.io,resources=schedulers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=install.armadaproject.io,resources=schedulers/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=groups;users,verbs=impersonate
//+kubebuilder:rbac:groups=core,resources=secrets;services;serviceaccounts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="batch",resources=jobs;cronjobs,verbs=get;list;watch;create;delete;deletecollection;patch;update
//+kubebuilder:rbac:groups=monitoring.coreos.com,resources=prometheusrules;servicemonitors,verbs=get;list;create;update;patch;delete
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles;clusterrolebindings;users,verbs=get;list;watch;create;update;patch;delete;impersonate

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *SchedulerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("namespace", req.Namespace, "name", req.Name)
	started := time.Now()
	logger.Info("Reconciling object")

	var scheduler installv1alpha1.Scheduler
	if miss, err := getObject(ctx, r.Client, &scheduler, req.NamespacedName, logger); err != nil || miss {
		return ctrl.Result{}, err
	}

	finish, err := checkAndHandleObjectDeletion(ctx, r.Client, &scheduler, operatorFinalizer, nil, logger)
	if err != nil || finish {
		return ctrl.Result{}, err
	}

	scheduler.Spec.PortConfig, err = installv1alpha1.BuildPortConfig(scheduler.Spec.ApplicationConfig)
	if err != nil {
		return ctrl.Result{}, err
	}

	var components *CommonComponents
	components, err = generateSchedulerInstallComponents(&scheduler, r.Scheme)
	if err != nil {
		return ctrl.Result{}, err
	}

	componentsCopy := components.DeepCopy()

	mutateFn := func() error {
		components.ReconcileComponents(componentsCopy)
		return nil
	}

	if err := upsertObjectIfNeeded(ctx, r.Client, components.ServiceAccount, scheduler.Kind, mutateFn, logger); err != nil {
		return ctrl.Result{}, err
	}

	if err := upsertObjectIfNeeded(ctx, r.Client, components.Secret, scheduler.Kind, mutateFn, logger); err != nil {
		return ctrl.Result{}, err
	}

	if len(components.Jobs) > 0 {
		if err := upsertObjectIfNeeded(ctx, r.Client, components.Jobs[0], scheduler.Kind, mutateFn, logger); err != nil {
			return ctrl.Result{}, err
		}

		if err := waitForJob(ctx, r.Client, components.Jobs[0], jobPollInterval, jobTimeout); err != nil {
			return ctrl.Result{}, err
		}
	}

	if err := upsertObjectIfNeeded(ctx, r.Client, components.Deployment, scheduler.Kind, mutateFn, logger); err != nil {
		return ctrl.Result{}, err
	}

	if err := upsertObjectIfNeeded(ctx, r.Client, components.Service, scheduler.Kind, mutateFn, logger); err != nil {
		return ctrl.Result{}, err
	}

	if err := upsertObjectIfNeeded(ctx, r.Client, components.IngressGrpc, scheduler.Kind, mutateFn, logger); err != nil {
		return ctrl.Result{}, err
	}

	if err := upsertObjectIfNeeded(ctx, r.Client, components.ServiceMonitor, scheduler.Kind, mutateFn, logger); err != nil {
		return ctrl.Result{}, err
	}

	if err := upsertObjectIfNeeded(ctx, r.Client, components.PrometheusRule, scheduler.Kind, mutateFn, logger); err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("Successfully reconciled Scheduler object", "durationMillis", time.Since(started).Milliseconds())

	return ctrl.Result{}, nil
}

type SchedulerConfig struct {
	Postgres PostgresConfig
}

func generateSchedulerInstallComponents(scheduler *installv1alpha1.Scheduler, scheme *runtime.Scheme) (*CommonComponents, error) {
	secret, err := builders.CreateSecret(scheduler.Spec.ApplicationConfig, scheduler.Name, scheduler.Namespace, GetConfigFilename(scheduler.Name))
	if err != nil {
		return nil, err
	}
	if err := controllerutil.SetOwnerReference(scheduler, secret, scheme); err != nil {
		return nil, err
	}

	var serviceAccount *corev1.ServiceAccount
	serviceAccountName := scheduler.Spec.CustomServiceAccount
	if serviceAccountName == "" {
		serviceAccount = builders.CreateServiceAccount(scheduler.Name, scheduler.Namespace, AllLabels(scheduler.Name, scheduler.Labels), scheduler.Spec.ServiceAccount)
		if err = controllerutil.SetOwnerReference(scheduler, serviceAccount, scheme); err != nil {
			return nil, errors.WithStack(err)
		}
		serviceAccountName = serviceAccount.Name
	}

	deployment, err := createSchedulerDeployment(scheduler, serviceAccountName)
	if err != nil {
		return nil, err
	}
	if err := controllerutil.SetOwnerReference(scheduler, deployment, scheme); err != nil {
		return nil, err
	}

	service := builders.Service(scheduler.Name, scheduler.Namespace, AllLabels(scheduler.Name, scheduler.Labels), IdentityLabel(scheduler.Name), scheduler.Spec.PortConfig)
	if err := controllerutil.SetOwnerReference(scheduler, service, scheme); err != nil {
		return nil, err
	}

	var serviceMonitor *monitoringv1.ServiceMonitor
	var prometheusRule *monitoringv1.PrometheusRule
	if scheduler.Spec.Prometheus != nil && scheduler.Spec.Prometheus.Enabled {
		serviceMonitor = createSchedulerServiceMonitor(scheduler)
		if err := controllerutil.SetOwnerReference(scheduler, serviceMonitor, scheme); err != nil {
			return nil, err
		}
		prometheusRule = createSchedulerPrometheusRule(scheduler)
		if err := controllerutil.SetOwnerReference(scheduler, prometheusRule, scheme); err != nil {
			return nil, err
		}
	}

	job, err := createSchedulerMigrationJob(scheduler, serviceAccountName)
	if err != nil {
		return nil, err
	}
	if err := controllerutil.SetOwnerReference(scheduler, job, scheme); err != nil {
		return nil, err
	}

	var cronJob *batchv1.CronJob
	if scheduler.Spec.Pruner != nil && scheduler.Spec.Pruner.Enabled {
		cronJob, err := createSchedulerCronJob(scheduler)
		if err != nil {
			return nil, err
		}
		if err := controllerutil.SetOwnerReference(scheduler, cronJob, scheme); err != nil {
			return nil, err
		}
	}

	ingressGrpc, err := createSchedulerIngressGrpc(scheduler)
	if err != nil {
		return nil, err
	}
	if ingressGrpc != nil {
		if err := controllerutil.SetOwnerReference(scheduler, ingressGrpc, scheme); err != nil {
			return nil, err
		}
	}

	return &CommonComponents{
		Deployment:     deployment,
		Service:        service,
		ServiceAccount: serviceAccount,
		Secret:         secret,
		IngressGrpc:    ingressGrpc,
		Jobs:           []*batchv1.Job{job},
		ServiceMonitor: serviceMonitor,
		PrometheusRule: prometheusRule,
		CronJob:        cronJob,
	}, nil
}

// Function to build the deployment object for Scheduler.
// This should be changing from CRD to CRD.  Not sure if generailize this helps much
func createSchedulerDeployment(scheduler *installv1alpha1.Scheduler, serviceAccountName string) (*appsv1.Deployment, error) {
	var runAsUser int64 = 1000
	var runAsGroup int64 = 2000
	allowPrivilegeEscalation := false
	env := createEnv(scheduler.Spec.Environment)
	pulsarConfig, err := ExtractPulsarConfig(scheduler.Spec.ApplicationConfig)
	if err != nil {
		return nil, err
	}
	volumes := createVolumes(scheduler.Name, scheduler.Spec.AdditionalVolumes)
	volumes = append(volumes, createPulsarVolumes(pulsarConfig)...)
	volumeMounts := createVolumeMounts(GetConfigFilename(scheduler.Name), scheduler.Spec.AdditionalVolumeMounts)
	volumeMounts = append(volumeMounts, createPulsarVolumeMounts(pulsarConfig)...)

	deployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: scheduler.Name, Namespace: scheduler.Namespace, Labels: AllLabels(scheduler.Name, scheduler.Labels)},
		Spec: appsv1.DeploymentSpec{
			Replicas: scheduler.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: IdentityLabel(scheduler.Name),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:        scheduler.Name,
					Namespace:   scheduler.Namespace,
					Labels:      AllLabels(scheduler.Name, scheduler.Labels),
					Annotations: map[string]string{"checksum/config": GenerateChecksumConfig(scheduler.Spec.ApplicationConfig.Raw)},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName:            serviceAccountName,
					TerminationGracePeriodSeconds: scheduler.DeletionGracePeriodSeconds,
					SecurityContext: &corev1.PodSecurityContext{
						RunAsUser:  &runAsUser,
						RunAsGroup: &runAsGroup,
					},
					Affinity: &corev1.Affinity{
						PodAntiAffinity: &corev1.PodAntiAffinity{
							PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{{
								Weight: 100,
								PodAffinityTerm: corev1.PodAffinityTerm{
									TopologyKey: "kubernetes.io/hostname",
									LabelSelector: &metav1.LabelSelector{
										MatchExpressions: []metav1.LabelSelectorRequirement{{
											Key:      "app",
											Operator: metav1.LabelSelectorOpIn,
											Values:   []string{scheduler.Name},
										}},
									},
								},
							}},
						},
					},
					Containers: []corev1.Container{{
						Name:            "scheduler",
						ImagePullPolicy: "IfNotPresent",
						Image:           ImageString(scheduler.Spec.Image),
						Args:            []string{"run", appConfigFlag, appConfigFilepath},
						Ports: []corev1.ContainerPort{
							{
								Name:          "metrics",
								ContainerPort: scheduler.Spec.PortConfig.MetricsPort,
								Protocol:      "TCP",
							},
							{
								Name:          "grpc",
								ContainerPort: scheduler.Spec.PortConfig.GrpcPort,
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

	if scheduler.Spec.Resources != nil {
		deployment.Spec.Template.Spec.Containers[0].Resources = *scheduler.Spec.Resources
		deployment.Spec.Template.Spec.Containers[0].Env = addGoMemLimit(deployment.Spec.Template.Spec.Containers[0].Env, *scheduler.Spec.Resources)
	}

	return &deployment, nil
}

func createSchedulerIngressGrpc(scheduler *installv1alpha1.Scheduler) (*networking.Ingress, error) {
	if len(scheduler.Spec.HostNames) == 0 {
		// when no hostnames provided, no ingress can be configured
		return nil, nil
	}
	ingressName := scheduler.Name + "-grpc"
	ingressHttp := &networking.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name: ingressName, Namespace: scheduler.Namespace, Labels: AllLabels(scheduler.Name, scheduler.Labels),
			Annotations: map[string]string{
				"kubernetes.io/ingress.class":                  scheduler.Spec.Ingress.IngressClass,
				"nginx.ingress.kubernetes.io/ssl-redirect":     "true",
				"nginx.ingress.kubernetes.io/backend-protocol": "GRPC",
			},
		},
	}

	if scheduler.Spec.ClusterIssuer != "" {
		ingressHttp.ObjectMeta.Annotations["certmanager.k8s.io/cluster-issuer"] = scheduler.Spec.ClusterIssuer
		ingressHttp.ObjectMeta.Annotations["cert-manager.io/cluster-issuer"] = scheduler.Spec.ClusterIssuer
	}

	if scheduler.Spec.Ingress.Annotations != nil {
		for key, value := range scheduler.Spec.Ingress.Annotations {
			ingressHttp.ObjectMeta.Annotations[key] = value
		}
	}
	ingressHttp.ObjectMeta.Labels = AllLabels(scheduler.Name, scheduler.Spec.Labels, scheduler.Spec.Ingress.Labels)

	secretName := scheduler.Name + "-service-tls"
	ingressHttp.Spec.TLS = []networking.IngressTLS{{Hosts: scheduler.Spec.HostNames, SecretName: secretName}}
	var ingressRules []networking.IngressRule
	serviceName := scheduler.Name
	for _, val := range scheduler.Spec.HostNames {
		ingressRules = append(ingressRules, networking.IngressRule{Host: val, IngressRuleValue: networking.IngressRuleValue{
			HTTP: &networking.HTTPIngressRuleValue{
				Paths: []networking.HTTPIngressPath{{
					Path:     "/",
					PathType: (*networking.PathType)(ptr.To[string]("Prefix")),
					Backend: networking.IngressBackend{
						Service: &networking.IngressServiceBackend{
							Name: serviceName,
							Port: networking.ServiceBackendPort{
								Number: scheduler.Spec.PortConfig.GrpcPort,
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

// createSchedulerMigrationJob returns a batch Job or an error if the app config is not correct
func createSchedulerMigrationJob(scheduler *installv1alpha1.Scheduler, serviceAccountName string) (*batchv1.Job, error) {
	runAsUser := int64(1000)
	runAsGroup := int64(2000)
	var terminationGracePeriodSeconds int64
	if scheduler.Spec.TerminationGracePeriodSeconds != nil {
		terminationGracePeriodSeconds = *scheduler.Spec.TerminationGracePeriodSeconds
	}
	allowPrivilegeEscalation := false
	parallelism := int32(1)
	completions := int32(1)
	backoffLimit := int32(0)
	env := scheduler.Spec.Environment
	volumes := createVolumes(scheduler.Name, scheduler.Spec.AdditionalVolumes)
	volumeMounts := createVolumeMounts(GetConfigFilename(scheduler.Name), scheduler.Spec.AdditionalVolumeMounts)

	appConfig, err := builders.ConvertRawExtensionToYaml(scheduler.Spec.ApplicationConfig)
	if err != nil {
		return nil, err
	}
	var schedulerConfig SchedulerConfig
	err = yaml.Unmarshal([]byte(appConfig), &schedulerConfig)
	if err != nil {
		return nil, err
	}

	job := batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:        scheduler.Name + "-migration",
			Namespace:   scheduler.Namespace,
			Labels:      AllLabels(scheduler.Name, scheduler.Labels),
			Annotations: map[string]string{"checksum/config": GenerateChecksumConfig(scheduler.Spec.ApplicationConfig.Raw)},
		},
		Spec: batchv1.JobSpec{
			Parallelism:  &parallelism,
			Completions:  &completions,
			BackoffLimit: &backoffLimit,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      scheduler.Name + "-migration",
					Namespace: scheduler.Namespace,
					Labels:    AllLabels(scheduler.Name, scheduler.Labels),
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
						Name:  "scheduler-migration-db-wait",
						Image: "postgres:15.2-alpine",
						Command: []string{
							"/bin/sh",
							"-c",
							`echo "Waiting for Postres..."
                                                         while ! nc -z $PGHOST $PGPORT; do
                                                           sleep 1
                                                         done
                                                         echo "Postgres started!"
							 echo "Creating DB $PGDB if needed..."
							 psql -v ON_ERROR_STOP=1 --username "$PGUSER" -c "CREATE DATABASE $PGDB"
							 psql -v ON_ERROR_STOP=1 --username "$PGUSER" -c "GRANT ALL PRIVILEGES ON DATABASE $PGDB TO $PGUSER"
							 echo "DB $PGDB created"`,
						},
						Env: []corev1.EnvVar{
							{
								Name:  "PGHOST",
								Value: schedulerConfig.Postgres.Connection.Host,
							},
							{
								Name:  "PGPORT",
								Value: schedulerConfig.Postgres.Connection.Port,
							},
							{
								Name:  "PGUSER",
								Value: schedulerConfig.Postgres.Connection.User,
							},
							{
								Name:  "PGPASSWORD",
								Value: schedulerConfig.Postgres.Connection.Password,
							},
							{
								Name:  "PGDB",
								Value: schedulerConfig.Postgres.Connection.Dbname,
							},
						},
					}},
					Containers: []corev1.Container{{
						Name:            "scheduler-migration",
						ImagePullPolicy: "IfNotPresent",
						Image:           ImageString(scheduler.Spec.Image),
						Args: []string{
							"migrateDatabase",
							appConfigFlag,
							appConfigFilepath,
						},
						Ports: []corev1.ContainerPort{{
							Name:          "metrics",
							ContainerPort: scheduler.Spec.PortConfig.MetricsPort,
							Protocol:      "TCP",
						}},
						Env:             env,
						VolumeMounts:    volumeMounts,
						SecurityContext: &corev1.SecurityContext{AllowPrivilegeEscalation: &allowPrivilegeEscalation},
					}},
					Tolerations: scheduler.Spec.Tolerations,
					Volumes:     volumes,
				},
			},
		},
	}

	return &job, nil
}

// createSchedulerCronJob returns a batch CronJob or an error if the app config is not correct
func createSchedulerCronJob(scheduler *installv1alpha1.Scheduler) (*batchv1.CronJob, error) {
	runAsUser := int64(1000)
	runAsGroup := int64(2000)
	terminationGracePeriodSeconds := int64(0)
	if scheduler.Spec.TerminationGracePeriodSeconds != nil {
		terminationGracePeriodSeconds = *scheduler.Spec.TerminationGracePeriodSeconds
	}
	allowPrivilegeEscalation := false
	parallelism := int32(1)
	completions := int32(1)
	backoffLimit := int32(0)
	env := scheduler.Spec.Environment
	volumes := createVolumes(scheduler.Name, scheduler.Spec.AdditionalVolumes)
	volumeMounts := createVolumeMounts(GetConfigFilename(scheduler.Name), scheduler.Spec.AdditionalVolumeMounts)

	appConfig, err := builders.ConvertRawExtensionToYaml(scheduler.Spec.ApplicationConfig)
	if err != nil {
		return nil, err
	}
	var schedulerConfig SchedulerConfig
	err = yaml.Unmarshal([]byte(appConfig), &schedulerConfig)
	if err != nil {
		return nil, err
	}

	prunerArgs := []string{
		"--pruneDatabase",
		appConfigFlag,
		appConfigFilepath,
	}
	if scheduler.Spec.Pruner.Args.Timeout != "" {
		prunerArgs = append(prunerArgs, "--timeout", scheduler.Spec.Pruner.Args.Timeout)
	}
	if scheduler.Spec.Pruner.Args.Batchsize > 0 {
		prunerArgs = append(prunerArgs, "--batchsize", fmt.Sprintf("%v", scheduler.Spec.Pruner.Args.Batchsize))
	}
	if scheduler.Spec.Pruner.Args.ExpireAfter != "" {
		prunerArgs = append(prunerArgs, "--expireAfter", scheduler.Spec.Pruner.Args.ExpireAfter)
	}
	prunerResources := corev1.ResourceRequirements{}
	if scheduler.Spec.Pruner.Resources != nil {
		prunerResources = *scheduler.Spec.Pruner.Resources
	}

	job := batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:        scheduler.Name + "-db-pruner",
			Namespace:   scheduler.Namespace,
			Labels:      AllLabels(scheduler.Name, scheduler.Labels),
			Annotations: map[string]string{"checksum/config": GenerateChecksumConfig(scheduler.Spec.ApplicationConfig.Raw)},
		},
		Spec: batchv1.CronJobSpec{

			Schedule: scheduler.Spec.Pruner.Schedule,
			JobTemplate: batchv1.JobTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      scheduler.Name + "-db-pruner",
					Namespace: scheduler.Namespace,
					Labels:    AllLabels(scheduler.Name, scheduler.Labels),
				},
				Spec: batchv1.JobSpec{
					Parallelism:  &parallelism,
					Completions:  &completions,
					BackoffLimit: &backoffLimit,
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Name:      scheduler.Name + "-db-pruner",
							Namespace: scheduler.Namespace,
							Labels:    AllLabels(scheduler.Name, scheduler.Labels),
						},
						Spec: corev1.PodSpec{
							RestartPolicy:                 "Never",
							TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
							SecurityContext: &corev1.PodSecurityContext{
								RunAsUser:  &runAsUser,
								RunAsGroup: &runAsGroup,
							},
							InitContainers: []corev1.Container{{
								Name:  "scheduler-db-pruner-db-wait",
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
										Value: schedulerConfig.Postgres.Connection.Host,
									},
									{
										Name:  "PGPORT",
										Value: schedulerConfig.Postgres.Connection.Port,
									},
								},
							}},
							Containers: []corev1.Container{{
								Name:            "scheduler-db-pruner",
								ImagePullPolicy: "IfNotPresent",
								Image:           ImageString(scheduler.Spec.Image),
								Args:            prunerArgs,
								Ports: []corev1.ContainerPort{{
									Name:          "metrics",
									ContainerPort: scheduler.Spec.PortConfig.MetricsPort,
									Protocol:      "TCP",
								}},
								Env:             env,
								VolumeMounts:    volumeMounts,
								SecurityContext: &corev1.SecurityContext{AllowPrivilegeEscalation: &allowPrivilegeEscalation},
								Resources:       prunerResources,
							}},
							Tolerations: scheduler.Spec.Tolerations,
							Volumes:     volumes,
						},
					},
				},
			},
		},
	}

	return &job, nil
}

// createSchedulerServiceMonitor will return a ServiceMonitor for this
func createSchedulerServiceMonitor(scheduler *installv1alpha1.Scheduler) *monitoringv1.ServiceMonitor {
	scrapeInterval := &metav1.Duration{Duration: defaultPrometheusInterval}
	if scheduler.Spec.Prometheus.ScrapeInterval == nil {
		scrapeInterval = &metav1.Duration{Duration: defaultPrometheusInterval}
	}
	durationString := duration.ShortHumanDuration(scrapeInterval.Duration)
	return &monitoringv1.ServiceMonitor{
		TypeMeta: metav1.TypeMeta{
			Kind: "ServiceMonitor",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      scheduler.Name,
			Namespace: scheduler.Namespace,
			Labels:    AllLabels(scheduler.Name, scheduler.Spec.Labels, scheduler.Spec.Prometheus.Labels),
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			Endpoints: []monitoringv1.Endpoint{
				{Port: "metrics", Interval: monitoringv1.Duration(durationString)},
			},
		},
	}
}

// createSchedulerPrometheusRule creates a PrometheusRule for monitoring Armada scheduler.
func createSchedulerPrometheusRule(scheduler *installv1alpha1.Scheduler) *monitoringv1.PrometheusRule {
	rules := []monitoringv1.Rule{
		{
			Record: "node:armada_scheduler_failed_jobs",
			Expr:   intstr.IntOrString{StrVal: `sum by (node) (armada_scheduler_job_state_counter_by_node{state="failed"})`},
		},
		{
			Record: "cluster_category_subCategory:armada_scheduler_failed_jobs",
			Expr:   intstr.IntOrString{StrVal: `sum by (cluster, category, subCategory) (armada_scheduler_error_classification_by_node)`},
		},
		{
			Record: "queue_category_subCategory:armada_scheduler_failed_jobs",
			Expr:   intstr.IntOrString{StrVal: `sum by (queue, category, subCategory) (armada_scheduler_job_error_classification_by_queue)`},
		},
		{
			Record: "node:armada_scheduler_succeeded_jobs",
			Expr:   intstr.IntOrString{StrVal: `sum by (node) (armada_scheduler_job_state_counter_by_node{state="succeeded"})`},
		},
		{
			Record: "cluster_category_subCategory:armada_scheduler_succeeded_jobs",
			Expr:   intstr.IntOrString{StrVal: `sum by (cluster, category, subCategory) (armada_scheduler_job_state_counter_by_node{state="succeeded"})`},
		},
		{
			Record: "queue_category_subCategory:armada_scheduler_succeeded_jobs",
			Expr:   intstr.IntOrString{StrVal: `sum by (queue) (armada_scheduler_job_state_counter_by_queue{state="succeeded"})`},
		},
		{
			Record: "node:armada_scheduler_failed_jobs:increase1m",
			Expr:   intstr.IntOrString{StrVal: `increase(node:armada_scheduler_job_state_counter_by_queue{state="failed"}[1m:])`},
		},
		{
			Record: "node:armada_scheduler_failed_jobs:increase10m",
			Expr:   intstr.IntOrString{StrVal: `increase(node:armada_scheduler_job_state_counter_by_queue{state="failed"}[10m:])`},
		},
		{
			Record: "node:armada_scheduler_failed_jobs:increase1h",
			Expr:   intstr.IntOrString{StrVal: `increase(node:armada_scheduler_job_state_counter_by_queue{state="failed"}[1h:])`},
		},
		{
			Record: "cluster_category_subCategory:armada_scheduler_failed_jobs:increase1m",
			Expr:   intstr.IntOrString{StrVal: `increase(cluster_category_subCategory:armada_scheduler_failed_jobs[1m:])`},
		},
		{
			Record: "cluster_category_subCategory:armada_scheduler_failed_jobs:increase10m",
			Expr:   intstr.IntOrString{StrVal: `increase(cluster_category_subCategory:armada_scheduler_failed_jobs[10m:])`},
		},
		{
			Record: "cluster_category_subCategory:armada_scheduler_failed_jobs:increase1h",
			Expr:   intstr.IntOrString{StrVal: `increase(cluster_category_subCategory:armada_scheduler_failed_jobs[1h:])`},
		},
		{
			Record: "queue_category_subCategory:armada_scheduler_failed_jobs:increase1m",
			Expr:   intstr.IntOrString{StrVal: `increase(queue_category_subCategory:armada_scheduler_failed_jobs[1m:])`},
		},
		{
			Record: "queue_category_subCategory:armada_scheduler_failed_jobs:increase10m",
			Expr:   intstr.IntOrString{StrVal: `increase(queue_category_subCategory:armada_scheduler_failed_jobs[10m:])`},
		},
		{
			Record: "queue_category_subCategory:armada_scheduler_failed_jobs:increase1h",
			Expr:   intstr.IntOrString{StrVal: `increase(queue_category_subCategory:armada_scheduler_failed_jobs[1h:])`},
		},
		{
			Record: "node:armada_scheduler_succeeded_jobs:increase1m",
			Expr:   intstr.IntOrString{StrVal: `increase(node:armada_scheduler_succeeded_jobs[1m:])`},
		},
		{
			Record: "node:armada_scheduler_succeeded_jobs:increase10m",
			Expr:   intstr.IntOrString{StrVal: `increase(node:armada_scheduler_succeeded_jobs[10m:])`},
		},
		{
			Record: "node:armada_scheduler_succeeded_jobs:increase1h",
			Expr:   intstr.IntOrString{StrVal: `increase(node:armada_scheduler_succeeded_jobs[1h:])`},
		},
		{
			Record: "cluster_category_subCategory:armada_scheduler_succeeded_jobs:increase1m",
			Expr:   intstr.IntOrString{StrVal: `increase(cluster_category_subCategory:armada_scheduler_succeeded_jobs[1m:])`},
		},
		{
			Record: "cluster_category_subCategory:armada_scheduler_succeeded_jobs:increase10m",
			Expr:   intstr.IntOrString{StrVal: `increase(cluster_category_subCategory:armada_scheduler_succeeded_jobs[10m:])`},
		},
		{
			Record: "cluster_category_subCategory:armada_scheduler_succeeded_jobs:increase1h",
			Expr:   intstr.IntOrString{StrVal: `increase(cluster_category_subCategory:armada_scheduler_succeeded_jobs[1h:])`},
		},
		{
			Record: "queue_category_subCategory:armada_scheduler_succeeded_jobs:increase1m",
			Expr:   intstr.IntOrString{StrVal: `increase(queue_category_subCategory:armada_scheduler_succeeded_jobs[1m:])`},
		},
		{
			Record: "queue_category_subCategory:armada_scheduler_succeeded_jobs:increase10m",
			Expr:   intstr.IntOrString{StrVal: `increase(queue_category_subCategory:armada_scheduler_succeeded_jobs[10m:])`},
		},
		{
			Record: "queue_category_subCategory:armada_scheduler_succeeded_jobs:increase1h",
			Expr:   intstr.IntOrString{StrVal: `increase(queue_category_subCategory:armada_scheduler_succeeded_jobs[1h:])`},
		},
		{
			Record: "node:armada_scheduler_failed_rate_jobs:increase1m",
			Expr:   intstr.IntOrString{StrVal: `sum by(node) (node:armada_scheduler_failed_jobs:increase1m) / on(node) group_left() ((sum by(node) (node:armada_scheduler_failed_jobs:increase1m)) + (sum by(node) (node:armada_scheduler_succeeded_jobs:increase1m)))`},
		},
		{
			Record: "node:armada_scheduler_failed_rate_jobs:increase10m",
			Expr:   intstr.IntOrString{StrVal: `sum by(node) (node:armada_scheduler_failed_jobs:increase10m) / on(node) group_left() ((sum by(node) (node:armada_scheduler_failed_jobs:increase10m)) + (sum by(node) (node:armada_scheduler_succeeded_jobs:increase10m)))`},
		},
		{
			Record: "node:armada_scheduler_failed_rate_jobs:increase1h",
			Expr:   intstr.IntOrString{StrVal: `sum by(node) (node:armada_scheduler_failed_jobs:increase1h) / on(node) group_left() ((sum by(node) (node:armada_scheduler_failed_jobs:increase1h)) + (sum by(node) (node:armada_scheduler_succeeded_jobs:increase1h)))`},
		},
		{
			Record: "cluster_category_subCategory:armada_scheduler_failed_rate_jobs:increase1m",
			Expr:   intstr.IntOrString{StrVal: `sum by(cluster, category, subCategory) (cluster_category_subCategory:armada_scheduler_failed_jobs:increase1m) / on(cluster) group_left() ((sum by(cluster) (cluster_category_subCategory:armada_scheduler_failed_jobs:increase1m)) + (sum by(cluster) (cluster_category_subCategory:armada_scheduler_succeeded_jobs:increase1m)))`},
		},
		{
			Record: "cluster_category_subCategory:armada_scheduler_failed_rate_jobs:increase10m",
			Expr:   intstr.IntOrString{StrVal: `sum by(cluster, category, subCategory) (cluster_category_subCategory:armada_scheduler_failed_jobs:increase10m) / on(cluster) group_left() ((sum by(cluster) (cluster_category_subCategory:armada_scheduler_failed_jobs:increase10m)) + (sum by(cluster) (cluster_category_subCategory:armada_scheduler_succeeded_jobs:increase10m)))`},
		},
		{
			Record: "cluster_category_subCategory:armada_scheduler_failed_rate_jobs:increase1h",
			Expr:   intstr.IntOrString{StrVal: `sum by(cluster, category, subCategory) (cluster_category_subCategory:armada_scheduler_failed_jobs:increase1h) / on(cluster) group_left() ((sum by(cluster) (cluster_category_subCategory:armada_scheduler_failed_jobs:increase1h)) + (sum by(cluster) (cluster_category_subCategory:armada_scheduler_succeeded_jobs:increase1h)))`},
		},
		{
			Record: "queue_category_subCategory:armada_scheduler_failed_rate_jobs:increase1m",
			Expr:   intstr.IntOrString{StrVal: `sum by(queue, category, subCategory) (queue_category_subCategory:armada_scheduler_failed_jobs:increase1m) / on(queue) group_left() ((sum by(queue) (queue_category_subCategory:armada_scheduler_failed_jobs:increase1m)) + (sum by(queue) (queue_category_subCategory:armada_scheduler_succeeded_jobs:increase1m)))`},
		},
		{
			Record: "queue_category_subCategory:armada_scheduler_failed_rate_jobs:increase10m",
			Expr:   intstr.IntOrString{StrVal: `sum by(queue, category, subCategory) (queue_category_subCategory:armada_scheduler_failed_jobs:increase10m) / on(queue) group_left() ((sum by(queue) (queue_category_subCategory:armada_scheduler_failed_jobs:increase10m)) + (sum by(queue) (queue_category_subCategory:armada_scheduler_succeeded_jobs:increase10m)))`},
		},
		{
			Record: "queue_category_subCategory:armada_scheduler_failed_rate_jobs:increase1h",
			Expr:   intstr.IntOrString{StrVal: `sum by(queue, category, subCategory) (queue_category_subCategory:armada_scheduler_failed_jobs:increase1h) / on(queue) group_left() ((sum by(queue) (queue_category_subCategory:armada_scheduler_failed_jobs:increase1h)) + (sum by(queue) (queue_category_subCategory:armada_scheduler_succeeded_jobs:increase1h)))`},
		},
	}

	objectMetaName := "armada-" + scheduler.Name + "-metrics"
	scrapeInterval := &metav1.Duration{Duration: defaultPrometheusInterval}
	if interval := scheduler.Spec.Prometheus.ScrapeInterval; interval != nil {
		scrapeInterval = &metav1.Duration{Duration: interval.Duration}
	}
	durationString := duration.ShortHumanDuration(scrapeInterval.Duration)
	return &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      scheduler.Name,
			Namespace: scheduler.Namespace,
			Labels:    AllLabels(scheduler.Name, scheduler.Spec.Labels, scheduler.Spec.Prometheus.Labels),
		},
		Spec: monitoringv1.PrometheusRuleSpec{
			Groups: []monitoringv1.RuleGroup{{
				Name:     objectMetaName,
				Interval: ptr.To(monitoringv1.Duration(durationString)),
				Rules:    rules,
			}},
		},
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *SchedulerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&installv1alpha1.Scheduler{}).
		Complete(r)
}
