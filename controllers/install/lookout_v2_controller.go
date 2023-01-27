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

	installv1alpha1 "github.com/armadaproject/armada-operator/apis/install/v1alpha1"
	"github.com/armadaproject/armada-operator/controllers/builders"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/yaml"
)

// LookoutV2Reconciler reconciles a LookoutV2 object
type LookoutV2Reconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=install.armadaproject.io,resources=lookoutV2,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=install.armadaproject.io,resources=lookoutV2/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=install.armadaproject.io,resources=lookoutV2/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *LookoutV2Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("namespace", req.Namespace, "name", req.Name)
	started := time.Now()
	logger.Info("Reconciling LookoutV2 object")

	logger.Info("Fetching LookoutV2 object from cache")

	var lookoutV2 installv1alpha1.LookoutV2
	if err := r.Client.Get(ctx, req.NamespacedName, &lookoutV2); err != nil {
		if k8serrors.IsNotFound(err) {
			logger.Info("LookoutV2 not found in cache, ending reconcile...", "namespace", req.Namespace, "name", req.Name)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	var components *LookoutV2Components
	components, err := generateLookoutV2InstallComponents(&lookoutV2, r.Scheme)
	if err != nil {
		return ctrl.Result{}, err
	}

	deletionTimestamp := lookoutV2.ObjectMeta.DeletionTimestamp
	// examine DeletionTimestamp to determine if object is under deletion
	if !deletionTimestamp.IsZero() {
		logger.Info("ArmadaServer object is being deleted")
		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}
	mutateFn := func() error { return nil }

	if components.ServiceAccount != nil {
		logger.Info("Upserting LookoutV2 ServiceAccount object")
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.ServiceAccount, mutateFn); err != nil {
			return ctrl.Result{}, err
		}
	}

	if components.Secret != nil {
		logger.Info("Upserting LookoutV2 Secret object")
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.Secret, mutateFn); err != nil {
			return ctrl.Result{}, err
		}
	}

	if components.Job != nil {
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.Job, mutateFn); err != nil {
			return ctrl.Result{}, err
		}
		ctxTimeout, cancel := context.WithTimeout(ctx, migrationTimeout)
		defer cancel()
		err := waitForJob(ctxTimeout, r.Client, components.Job, migrationPollSleep)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	if components.Deployment != nil {
		logger.Info("Upserting LookoutV2 Deployment object")
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.Deployment, mutateFn); err != nil {
			return ctrl.Result{}, err
		}
	}

	if components.Service != nil {
		logger.Info("Upserting LookoutV2 Service object")
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.Service, mutateFn); err != nil {
			return ctrl.Result{}, err
		}
	}

	if components.IngressWeb != nil {
		logger.Info("Upserting LookoutV2 Ingress object")
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.IngressWeb, mutateFn); err != nil {
			return ctrl.Result{}, err
		}
	}

	logger.Info("Successfully reconciled LookoutV2 object", "durationMilis", time.Since(started).Milliseconds())

	return ctrl.Result{}, nil
}

type LookoutV2Components struct {
	Deployment     *appsv1.Deployment
	IngressWeb     *networking.Ingress
	Secret         *corev1.Secret
	Service        *corev1.Service
	ServiceAccount *corev1.ServiceAccount

	// For Migration
	Job *batchv1.Job
}

type LookoutV2Config struct {
	Postgres PostgresConfig
}

func generateLookoutV2InstallComponents(lookoutV2 *installv1alpha1.LookoutV2, scheme *runtime.Scheme) (*LookoutV2Components, error) {
	// serviceAccount := r.createServiceAccount(lookoutV2)
	// if err := controllerutil.SetOwnerReference(lookoutV2, serviceAccount, scheme); err != nil {
	// 	return nil, err
	// }
	secret, err := builders.CreateSecret(lookoutV2.Spec.ApplicationConfig, lookoutV2.Name, lookoutV2.Namespace, GetConfigFilename(lookoutV2.Name))
	if err != nil {
		return nil, err
	}
	if err := controllerutil.SetOwnerReference(lookoutV2, secret, scheme); err != nil {
		return nil, err
	}
	deployment := createLookoutV2Deployment(lookoutV2)
	if err := controllerutil.SetOwnerReference(lookoutV2, deployment, scheme); err != nil {
		return nil, err
	}
	service := builders.Service(lookoutV2.Name, lookoutV2.Namespace, AllLabels(lookoutV2.Name, lookoutV2.Labels))
	if err := controllerutil.SetOwnerReference(lookoutV2, service, scheme); err != nil {
		return nil, err
	}
	// serviceAccount := r.createServiceAccount(lookoutV2)
	// if err := controllerutil.SetOwnerReference(lookoutV2, serviceAccount, scheme); err != nil {
	// 	return nil, err
	// }
	var job *batchv1.Job
	if lookoutV2.Spec.MigrateDatabase {
		job, err = createLookoutV2MigrationJob(lookoutV2)
		if err != nil {
			return nil, err
		}
		if err := controllerutil.SetOwnerReference(lookoutV2, job, scheme); err != nil {
			return nil, err
		}
	}

	ingressWeb := createLookoutV2IngressWeb(lookoutV2)
	if err := controllerutil.SetOwnerReference(lookoutV2, ingressWeb, scheme); err != nil {
		return nil, err
	}

	return &LookoutV2Components{
		Deployment:     deployment,
		Service:        service,
		ServiceAccount: nil,
		Secret:         secret,
		IngressWeb:     ingressWeb,
		Job:            job,
	}, nil
}

// Function to build the deployment object for LookoutV2.
// This should be changing from CRD to CRD.  Not sure if generailize this helps much
func createLookoutV2Deployment(lookoutV2 *installv1alpha1.LookoutV2) *appsv1.Deployment {
	var replicas int32 = 1
	var runAsUser int64 = 1000
	var runAsGroup int64 = 2000
	allowPrivilegeEscalation := false

	deployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: lookoutV2.Name, Namespace: lookoutV2.Namespace, Labels: AllLabels(lookoutV2.Name, lookoutV2.Labels)},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: IdentityLabel(lookoutV2.Name),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:        lookoutV2.Name,
					Namespace:   lookoutV2.Namespace,
					Labels:      AllLabels(lookoutV2.Name, lookoutV2.Labels),
					Annotations: map[string]string{"checksum/config": GenerateChecksumConfig(lookoutV2.Spec.ApplicationConfig.Raw)},
				},
				Spec: corev1.PodSpec{
					TerminationGracePeriodSeconds: lookoutV2.DeletionGracePeriodSeconds,
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
											Values:   []string{lookoutV2.Name},
										}},
									},
								},
							}},
						},
					},
					Containers: []corev1.Container{{
						Name:            "lookoutV2",
						ImagePullPolicy: "IfNotPresent",
						Image:           ImageString(lookoutV2.Spec.Image),
						Args:            []string{"--config", "/config/application_config.yaml"},
						Ports: []corev1.ContainerPort{{
							Name:          "metrics",
							ContainerPort: 9001,
							Protocol:      "TCP",
						}},
						Env: []corev1.EnvVar{
							{
								Name: "SERVICE_ACCOUNT",
								ValueFrom: &corev1.EnvVarSource{
									FieldRef: &corev1.ObjectFieldSelector{
										FieldPath: "spec.serviceAccountName",
									},
								},
							},
							{
								Name: "POD_NAMESPACE",
								ValueFrom: &corev1.EnvVarSource{
									FieldRef: &corev1.ObjectFieldSelector{
										FieldPath: "metadata.namespace",
									},
								},
							},
						},
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      volumeConfigKey,
								ReadOnly:  true,
								MountPath: "/config/application_config.yaml",
								SubPath:   lookoutV2.Name,
							},
						},
						SecurityContext: &corev1.SecurityContext{AllowPrivilegeEscalation: &allowPrivilegeEscalation},
					}},
					Volumes: []corev1.Volume{{
						Name: volumeConfigKey,
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName: lookoutV2.Name,
							},
						},
					}},
				},
			},
		},
	}
	if lookoutV2.Spec.Resources != nil {
		deployment.Spec.Template.Spec.Containers[0].Resources = *lookoutV2.Spec.Resources
	}
	return &deployment
}

func createLookoutV2IngressWeb(lookoutV2 *installv1alpha1.LookoutV2) *networking.Ingress {
	ingressWeb := &networking.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: lookoutV2.Name, Namespace: lookoutV2.Namespace, Labels: AllLabels(lookoutV2.Name, lookoutV2.Labels),
			Annotations: map[string]string{
				"kubernetes.io/ingress.class":                lookoutV2.Spec.Ingress.IngressClass,
				"certmanager.k8s.io/cluster-issuer":          lookoutV2.Spec.ClusterIssuer,
				"cert-manager.io/cluster-issuer":             lookoutV2.Spec.ClusterIssuer,
				"nginx.ingress.kubernetes.io/rewrite-target": "/$2",
				"nginx.ingress.kubernetes.io/ssl-redirect":   "true",
			},
		},
	}

	if lookoutV2.Spec.Ingress.Annotations != nil {
		for key, value := range lookoutV2.Spec.Ingress.Annotations {
			ingressWeb.ObjectMeta.Annotations[key] = value
		}
	}
	if lookoutV2.Spec.Ingress.Labels != nil {
		for key, value := range lookoutV2.Spec.Ingress.Labels {
			ingressWeb.ObjectMeta.Labels[key] = value
		}
	}
	if len(lookoutV2.Spec.HostNames) > 0 {
		secretName := lookoutV2.Name + "-service-tls"
		ingressWeb.Spec.TLS = []networking.IngressTLS{{Hosts: lookoutV2.Spec.HostNames, SecretName: secretName}}
		ingressRules := []networking.IngressRule{}
		serviceName := "armada" + "-" + lookoutV2.Name
		for _, val := range lookoutV2.Spec.HostNames {
			ingressRules = append(ingressRules, networking.IngressRule{Host: val, IngressRuleValue: networking.IngressRuleValue{
				HTTP: &networking.HTTPIngressRuleValue{
					Paths: []networking.HTTPIngressPath{{
						Path:     "/api(/|$)(.*)",
						PathType: (*networking.PathType)(pointer.String("ImplementationSpecific")),
						Backend: networking.IngressBackend{
							Service: &networking.IngressServiceBackend{
								Name: serviceName,
								Port: networking.ServiceBackendPort{
									Number: 8081,
								},
							},
						},
					}},
				},
			}})
		}
		ingressWeb.Spec.Rules = ingressRules
	}
	return ingressWeb
}

func createLookoutV2MigrationJob(lookoutV2 *installv1alpha1.LookoutV2) (*batchv1.Job, error) {
	runAsUser := int64(1000)
	runAsGroup := int64(2000)
	terminationGracePeriodSeconds := int64(lookoutV2.Spec.TerminationGracePeriodSeconds)
	allowPrivilegeEscalation := false
	parallelism := int32(1)
	completions := int32(1)
	backoffLimit := int32(0)

	appConfig, err := builders.ConvertRawExtensionToYaml(lookoutV2.Spec.ApplicationConfig)
	if err != nil {
		return nil, err
	}
	var lookoutV2Config LookoutV2Config
	err = yaml.Unmarshal([]byte(appConfig), &lookoutV2Config)
	if err != nil {
		return nil, err
	}

	job := batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:        lookoutV2.Name + "-migration",
			Namespace:   lookoutV2.Namespace,
			Labels:      AllLabels(lookoutV2.Name, lookoutV2.Labels),
			Annotations: map[string]string{"checksum/config": GenerateChecksumConfig(lookoutV2.Spec.ApplicationConfig.Raw)},
		},
		Spec: batchv1.JobSpec{
			Parallelism:  &parallelism,
			Completions:  &completions,
			BackoffLimit: &backoffLimit,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      lookoutV2.Name + "-migration",
					Namespace: lookoutV2.Namespace,
					Labels:    AllLabels(lookoutV2.Name, lookoutV2.Labels),
				},
				Spec: corev1.PodSpec{
					RestartPolicy:                 "Never",
					TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
					SecurityContext: &corev1.PodSecurityContext{
						RunAsUser:  &runAsUser,
						RunAsGroup: &runAsGroup,
					},
					InitContainers: []corev1.Container{{
						Name:  "lookoutV2-migration-db-wait",
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
								Value: lookoutV2Config.Postgres.Connection.Host,
							},
							{
								Name:  "PGPORT",
								Value: lookoutV2Config.Postgres.Connection.Port,
							},
						},
					}},
					Containers: []corev1.Container{{
						Name:            "lookoutV2-migration",
						ImagePullPolicy: "IfNotPresent",
						Image:           ImageString(lookoutV2.Spec.Image),
						Args: []string{
							"--migrateDatabase",
							"--config",
							"/config/application_config.yaml",
						},
						Ports: []corev1.ContainerPort{{
							Name:          "metrics",
							ContainerPort: 9001,
							Protocol:      "TCP",
						}},
						Env: []corev1.EnvVar{
							{
								Name: "SERVICE_ACCOUNT",
								ValueFrom: &corev1.EnvVarSource{
									FieldRef: &corev1.ObjectFieldSelector{
										FieldPath: "spec.serviceAccountName",
									},
								},
							},
							{
								Name: "POD_NAMESPACE",
								ValueFrom: &corev1.EnvVarSource{
									FieldRef: &corev1.ObjectFieldSelector{
										FieldPath: "metadata.namespace",
									},
								},
							},
						},
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      volumeConfigKey,
								ReadOnly:  true,
								MountPath: "/config/application_config.yaml",
								SubPath:   lookoutV2.Name,
							},
						},
						SecurityContext: &corev1.SecurityContext{AllowPrivilegeEscalation: &allowPrivilegeEscalation},
					}},
					NodeSelector: lookoutV2.Spec.NodeSelector,
					Tolerations:  lookoutV2.Spec.Tolerations,
					Volumes: []corev1.Volume{{
						Name: volumeConfigKey,
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName: lookoutV2.Name,
							},
						},
					}},
				},
			},
		},
	}

	return &job, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *LookoutV2Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&installv1alpha1.LookoutV2{}).
		Complete(r)
}
