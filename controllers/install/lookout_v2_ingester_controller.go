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
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	installv1alpha1 "github.com/armadaproject/armada-operator/apis/install/v1alpha1"
	"github.com/armadaproject/armada-operator/controllers/builders"
)

// LookoutV2IngesterReconciler reconciles a LookoutV2Ingester object
type LookoutV2IngesterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=install.armadaproject.io,resources=lookoutV2ingesters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=install.armadaproject.io,resources=lookoutV2ingesters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=install.armadaproject.io,resources=lookoutV2ingesters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *LookoutV2IngesterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("namespace", req.Namespace, "name", req.Name)
	started := time.Now()
	logger.Info("Reconciling LookoutV2Ingester object")

	logger.Info("Fetching LookoutV2Ingester object from cache")
	var lookoutV2Ingester installv1alpha1.LookoutV2Ingester
	if err := r.Client.Get(ctx, req.NamespacedName, &lookoutV2Ingester); err != nil {
		if k8serrors.IsNotFound(err) {
			logger.Info("LookoutV2Ingester not found in cache, ending reconcile")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	logger.Info(fmt.Sprintf("LookoutV2Ingester Name %s", lookoutV2Ingester.Name))
	components, err := r.generateInstallComponents(&lookoutV2Ingester)
	if err != nil {
		return ctrl.Result{}, err
	}

	deletionTimestamp := lookoutV2Ingester.ObjectMeta.DeletionTimestamp

	// examine DeletionTimestamp to determine if object is under deletion
	if !deletionTimestamp.IsZero() {
		logger.Info("LookoutV2Ingester is being deleted")

		// FIXME: Seems like something actually has to happen here?

		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil

	}

	mutateFn := func() error {
		return nil
	}

	if components.Secret != nil {
		logger.Info("Upserting LookoutV2Ingester Secret object")
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.Secret, mutateFn); err != nil {
			return ctrl.Result{}, err
		}
	}

	if components.ServiceAccount != nil {
		logger.Info("Upserting LookoutV2Ingester ServiceAccount object")
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.ServiceAccount, mutateFn); err != nil {
			return ctrl.Result{}, err
		}
	}

	if components.Deployment != nil {
		logger.Info("Upserting LookoutV2Ingester Deployment object")
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.Deployment, mutateFn); err != nil {
			return ctrl.Result{}, err
		}
	}

	logger.Info("Successfully reconciled LookoutV2Ingester object", "durationMilis", time.Since(started).Milliseconds())

	return ctrl.Result{}, nil
}

type LookoutV2IngesterComponents struct {
	Deployment     *appsv1.Deployment
	ServiceAccount *corev1.ServiceAccount
	Secret         *corev1.Secret
}

func (ec *LookoutV2IngesterComponents) DeepCopy() *LookoutV2IngesterComponents {
	return &LookoutV2IngesterComponents{
		Deployment:     ec.Deployment.DeepCopy(),
		ServiceAccount: ec.ServiceAccount.DeepCopy(),
		Secret:         ec.Secret.DeepCopy(),
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *LookoutV2IngesterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&installv1alpha1.LookoutV2Ingester{}).
		Complete(r)
}

func (r *LookoutV2IngesterReconciler) generateInstallComponents(lookoutV2Ingester *installv1alpha1.LookoutV2Ingester) (*LookoutV2IngesterComponents, error) {
	secret, err := builders.CreateSecret(lookoutV2Ingester.Spec.ApplicationConfig, lookoutV2Ingester.Name, lookoutV2Ingester.Namespace, GetConfigFilename(lookoutV2Ingester.Name))
	if err != nil {
		return nil, err
	}
	if err := controllerutil.SetOwnerReference(lookoutV2Ingester, secret, r.Scheme); err != nil {
		return nil, err
	}
	deployment := r.createDeployment(lookoutV2Ingester, secret)
	if err := controllerutil.SetOwnerReference(lookoutV2Ingester, deployment, r.Scheme); err != nil {
		return nil, err
	}
	serviceAccount := builders.CreateServiceAccount(lookoutV2Ingester.Name, lookoutV2Ingester.Namespace, AllLabels(lookoutV2Ingester.Name, lookoutV2Ingester.Labels), lookoutV2Ingester.Spec.ServiceAccount)
	if err := controllerutil.SetOwnerReference(lookoutV2Ingester, serviceAccount, r.Scheme); err != nil {
		return nil, err
	}

	return &LookoutV2IngesterComponents{
		Deployment:     deployment,
		ServiceAccount: serviceAccount,
		Secret:         secret,
	}, nil
}

// TODO: Flesh this out for lookoutV2ingester
func (r *LookoutV2IngesterReconciler) createDeployment(lookoutV2Ingester *installv1alpha1.LookoutV2Ingester, secret *corev1.Secret) *appsv1.Deployment {
	var replicas int32 = 1
	var runAsUser int64 = 1000
	var runAsGroup int64 = 2000
	allowPrivilegeEscalation := false
	deployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: lookoutV2Ingester.Name, Namespace: lookoutV2Ingester.Namespace, Labels: AllLabels(lookoutV2Ingester.Name, lookoutV2Ingester.Labels)},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: IdentityLabel(lookoutV2Ingester.Name),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:        lookoutV2Ingester.Name,
					Namespace:   lookoutV2Ingester.Namespace,
					Labels:      AllLabels(lookoutV2Ingester.Name, lookoutV2Ingester.Labels),
					Annotations: map[string]string{"checksum/config": GenerateChecksumConfig(lookoutV2Ingester.Spec.ApplicationConfig.Raw)},
				},
				Spec: corev1.PodSpec{
					TerminationGracePeriodSeconds: lookoutV2Ingester.Spec.TerminationGracePeriodSeconds,
					SecurityContext: &corev1.PodSecurityContext{
						RunAsUser:  &runAsUser,
						RunAsGroup: &runAsGroup,
					},
					Containers: []corev1.Container{{
						Name:            "lookoutV2ingester",
						ImagePullPolicy: "IfNotPresent",
						Image:           ImageString(lookoutV2Ingester.Spec.Image),
						Args:            []string{"--config", "/config/application_config.yaml"},
						// FIXME(Clif): Needs to change
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
								SubPath:   lookoutV2Ingester.Name,
							},
						},
						SecurityContext: &corev1.SecurityContext{AllowPrivilegeEscalation: &allowPrivilegeEscalation},
					}},
					Tolerations: lookoutV2Ingester.Spec.Tolerations,
					Volumes: []corev1.Volume{{
						Name: volumeConfigKey,
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName: secret.Name,
							},
						},
					}},
				},
			},
		},
	}
	if lookoutV2Ingester.Spec.Resources != nil {
		deployment.Spec.Template.Spec.Containers[0].Resources = *lookoutV2Ingester.Spec.Resources
	}
	return &deployment
}
