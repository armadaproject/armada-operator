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
)

// LookoutReconciler reconciles a Lookout object
type LookoutReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=install.armadaproject.io,resources=lookout,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=install.armadaproject.io,resources=lookout/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=install.armadaproject.io,resources=lookout/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *LookoutReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("namespace", req.Namespace, "name", req.Name)
	started := time.Now()
	logger.Info("Reconciling Lookout object")

	logger.Info("Fetching Lookout object from cache")

	var lookout installv1alpha1.Lookout
	if err := r.Client.Get(ctx, req.NamespacedName, &lookout); err != nil {
		if k8serrors.IsNotFound(err) {
			logger.Info("Lookout not found in cache, ending reconcile...", "namespace", req.Namespace, "name", req.Name)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	var components *LookoutComponents
	components, err := generateLookoutInstallComponents(&lookout, r.Scheme)
	if err != nil {
		return ctrl.Result{}, err
	}

	deletionTimestamp := lookout.ObjectMeta.DeletionTimestamp
	// examine DeletionTimestamp to determine if object is under deletion
	if deletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !controllerutil.ContainsFinalizer(&lookout, operatorFinalizer) {
			logger.Info("Attaching finalizer to Lookout object", "finalizer", operatorFinalizer)
			controllerutil.AddFinalizer(&lookout, operatorFinalizer)
			if err := r.Update(ctx, &lookout); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		logger.Info("Lookout object is being deleted", "finalizer", operatorFinalizer)
		// The object is being deleted
		if controllerutil.ContainsFinalizer(&lookout, operatorFinalizer) {

			// remove our finalizer from the list and update it.
			logger.Info("Removing finalizer from Lookout object", "finalizer", operatorFinalizer)
			controllerutil.RemoveFinalizer(&lookout, operatorFinalizer)
			if err := r.Update(ctx, &lookout); err != nil {
				return ctrl.Result{}, err
			}
		}

		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}
	mutateFn := func() error { return nil }

	if components.ServiceAccount != nil {
		logger.Info("Upserting Lookout ServiceAccount object")
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.ServiceAccount, mutateFn); err != nil {
			return ctrl.Result{}, err
		}
	}

	if components.Secret != nil {
		logger.Info("Upserting Lookout Secret object")
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.Secret, mutateFn); err != nil {
			return ctrl.Result{}, err
		}
	}

	if components.Deployment != nil {
		logger.Info("Upserting Lookout Deployment object")
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.Deployment, mutateFn); err != nil {
			return ctrl.Result{}, err
		}
	}

	if components.Service != nil {
		logger.Info("Upserting Lookout Service object")
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.Service, mutateFn); err != nil {
			return ctrl.Result{}, err
		}
	}

	logger.Info("Successfully reconciled Lookout object", "durationMilis", time.Since(started).Milliseconds())

	return ctrl.Result{}, nil
}

type LookoutComponents struct {
	Deployment     *appsv1.Deployment
	IngressWeb     *networking.Ingress
	Secret         *corev1.Secret
	Service        *corev1.Service
	ServiceAccount *corev1.ServiceAccount

	// ToDo: add other components
	// CronJob        *batchv1beta1.CronJob
}

func generateLookoutInstallComponents(lookout *installv1alpha1.Lookout, scheme *runtime.Scheme) (*LookoutComponents, error) {
	secret, err := builders.CreateSecret(lookout.Spec.ApplicationConfig, lookout.Name, lookout.Namespace, GetConfigFilename(lookout.Name))
	if err != nil {
		return nil, err
	}
	if err := controllerutil.SetOwnerReference(lookout, secret, scheme); err != nil {
		return nil, err
	}
	deployment := createLookoutDeployment(lookout)
	if err := controllerutil.SetOwnerReference(lookout, deployment, scheme); err != nil {
		return nil, err
	}
	service := builders.Service(lookout.Name, lookout.Namespace, AllLabels(lookout.Name, lookout.Labels))
	if err := controllerutil.SetOwnerReference(lookout, service, scheme); err != nil {
		return nil, err
	}

	ingressWeb := createLookoutIngressWeb(lookout)
	if err := controllerutil.SetOwnerReference(lookout, ingressWeb, scheme); err != nil {
		return nil, err
	}

	return &LookoutComponents{
		Deployment:     deployment,
		Service:        service,
		ServiceAccount: nil,
		Secret:         secret,
		IngressWeb:     ingressWeb,
	}, nil
}

// Function to build the deployment object for Lookout.
// This should be changing from CRD to CRD.  Not sure if generailize this helps much
func createLookoutDeployment(lookout *installv1alpha1.Lookout) *appsv1.Deployment {
	var replicas int32 = 1
	var runAsUser int64 = 1000
	var runAsGroup int64 = 2000
	allowPrivilegeEscalation := false

	deployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: lookout.Name, Namespace: lookout.Namespace, Labels: AllLabels(lookout.Name, lookout.Labels)},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
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
								SubPath:   lookout.Name,
							},
						},
						SecurityContext: &corev1.SecurityContext{AllowPrivilegeEscalation: &allowPrivilegeEscalation},
					}},
					Volumes: []corev1.Volume{{
						Name: volumeConfigKey,
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName: lookout.Name,
							},
						},
					}},
				},
			},
		},
	}
	if lookout.Spec.Resources != nil {
		deployment.Spec.Template.Spec.Containers[0].Resources = *lookout.Spec.Resources
	}
	return &deployment
}

func createLookoutIngressWeb(lookout *installv1alpha1.Lookout) *networking.Ingress {
	ingressWeb := &networking.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: lookout.Name, Namespace: lookout.Namespace, Labels: AllLabels(lookout.Name, lookout.Labels),
			Annotations: map[string]string{
				"kubernetes.io/ingress.class":                lookout.Spec.Ingress.IngressClass,
				"certmanager.k8s.io/cluster-issuer":          lookout.Spec.ClusterIssuer,
				"cert-manager.io/cluster-issuer":             lookout.Spec.ClusterIssuer,
				"nginx.ingress.kubernetes.io/rewrite-target": "/$2",
				"nginx.ingress.kubernetes.io/ssl-redirect":   "true",
			},
		},
	}

	if lookout.Spec.Ingress.Annotations != nil {
		for key, value := range lookout.Spec.Ingress.Annotations {
			ingressWeb.ObjectMeta.Annotations[key] = value
		}
	}
	if lookout.Spec.Ingress.Labels != nil {
		for key, value := range lookout.Spec.Ingress.Labels {
			ingressWeb.ObjectMeta.Labels[key] = value
		}
	}
	if len(lookout.Spec.HostNames) > 0 {
		secretName := lookout.Name + "-service-tls"
		ingressWeb.Spec.TLS = []networking.IngressTLS{{Hosts: lookout.Spec.HostNames, SecretName: secretName}}
		ingressRules := []networking.IngressRule{}
		serviceName := "armada" + "-" + lookout.Name
		for _, val := range lookout.Spec.HostNames {
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

// SetupWithManager sets up the controller with the Manager.
func (r *LookoutReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&installv1alpha1.Lookout{}).
		Complete(r)
}
