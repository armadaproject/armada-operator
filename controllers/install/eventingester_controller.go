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
	"crypto/sha256"
	"encoding/hex"
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

const (
	eventIngesterVolumeConfigKey = "user-config"
)

// EventIngesterReconciler reconciles a EventIngester object
type EventIngesterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=install.armadaproject.io,resources=eventIngesters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=install.armadaproject.io,resources=eventIngesters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=install.armadaproject.io,resources=eventIngesters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the EventIngester object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *EventIngesterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("namespace", req.Namespace, "name", req.Name)
	started := time.Now()
	logger.Info("Reconciling EventIngester object")

	logger.Info("Fetching EventIngester object from cache")
	var eventIngester installv1alpha1.EventIngester
	if err := r.Client.Get(ctx, req.NamespacedName, &eventIngester); err != nil {
		if k8serrors.IsNotFound(err) {
			logger.Info("EventIngester not found in cache, ending reconcile")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	components, err := r.generateEventIngesterComponents(&eventIngester, r.Scheme)
	if err != nil {
		return ctrl.Result{}, err
	}

	deletionTimestamp := eventIngester.ObjectMeta.DeletionTimestamp
	// examine DeletionTimestamp to determine if object is under deletion
	if !deletionTimestamp.IsZero() {
		logger.Info("EventIngester is being deleted")

		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	mutateFn := func() error { return nil }

	if components.ServiceAccount != nil {
		logger.Info("Upserting EventIngester ServiceAccount object")
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.ServiceAccount, mutateFn); err != nil {
			return ctrl.Result{}, err
		}
	}

	if components.Secret != nil {
		logger.Info("Upserting EventIngester Secret object")
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.Secret, mutateFn); err != nil {
			return ctrl.Result{}, err
		}
	}

	if components.Deployment != nil {
		logger.Info("Upserting EventIngester Deployment object")
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.Deployment, mutateFn); err != nil {
			return ctrl.Result{}, err
		}
	}

	logger.Info("Successfully reconciled EventIngester object", "durationMilis", time.Since(started).Milliseconds())

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EventIngesterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&installv1alpha1.EventIngester{}).
		Complete(r)
}

type EventIngesterComponents struct {
	Deployment     *appsv1.Deployment
	ServiceAccount *corev1.ServiceAccount
	Secret         *corev1.Secret
}

func (r *EventIngesterReconciler) generateEventIngesterComponents(eventIngester *installv1alpha1.EventIngester, scheme *runtime.Scheme) (*EventIngesterComponents, error) {
	secret, err := builders.CreateSecret(eventIngester.Spec.ApplicationConfig, eventIngester.Name, eventIngester.Namespace)
	if err != nil {
		return nil, err
	}
	if err := controllerutil.SetOwnerReference(eventIngester, secret, scheme); err != nil {
		return nil, err
	}
	deployment := r.createDeployment(eventIngester)
	if err := controllerutil.SetOwnerReference(eventIngester, deployment, scheme); err != nil {
		return nil, err
	}

	return &EventIngesterComponents{
		Deployment:     deployment,
		ServiceAccount: nil,
		Secret:         secret,
	}, nil
}

func (r *EventIngesterReconciler) createDeployment(eventIngester *installv1alpha1.EventIngester) *appsv1.Deployment {
	var replicas int32 = 1
	var runAsUser int64 = 1000
	var runAsGroup int64 = 2000
	allowPrivilegeEscalation := false
	deployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: eventIngester.Name, Namespace: eventIngester.Namespace, Labels: getAllEventIngesterLabels(eventIngester)},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: getEventIngesterIdentityLabels(eventIngester),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:        eventIngester.Name,
					Namespace:   eventIngester.Namespace,
					Labels:      getAllEventIngesterLabels(eventIngester),
					Annotations: map[string]string{"checksum/config": getEventIngesterChecksumConfig(eventIngester)},
				},
				Spec: corev1.PodSpec{
					TerminationGracePeriodSeconds: eventIngester.Spec.TerminationGracePeriodSeconds,
					SecurityContext: &corev1.PodSecurityContext{
						RunAsUser:  &runAsUser,
						RunAsGroup: &runAsGroup,
					},
					Containers: []corev1.Container{{
						Name:            "eventIngester",
						ImagePullPolicy: "IfNotPresent",
						Image:           ImageString(eventIngester.Spec.Image),
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
								Name:      eventIngesterVolumeConfigKey,
								ReadOnly:  true,
								MountPath: "/config/application_config.yaml",
								SubPath:   getEventIngesterConfigFilename(eventIngester),
							},
						},
						SecurityContext: &corev1.SecurityContext{AllowPrivilegeEscalation: &allowPrivilegeEscalation},
					}},
					NodeSelector: eventIngester.Spec.NodeSelector,
					Tolerations:  eventIngester.Spec.Tolerations,
					Volumes: []corev1.Volume{{
						Name: eventIngesterVolumeConfigKey,
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName: getEventIngesterConfigName(eventIngester),
							},
						},
					}},
				},
			},
		},
	}
	if eventIngester.Spec.Resources != nil {
		deployment.Spec.Template.Spec.Containers[0].Resources = *eventIngester.Spec.Resources
	}
	return &deployment

}

func getEventIngesterConfigFilename(eventIngester *installv1alpha1.EventIngester) string {
	return getEventIngesterConfigName(eventIngester) + ".yaml"
}

func getEventIngesterConfigName(eventIngester *installv1alpha1.EventIngester) string {
	return fmt.Sprintf("%s-%s", eventIngester.Name, "config")
}

func getEventIngesterChecksumConfig(eventIngester *installv1alpha1.EventIngester) string {
	data := eventIngester.Spec.ApplicationConfig.Raw
	sha := sha256.Sum256(data)
	return hex.EncodeToString(sha[:])
}

func getAllEventIngesterLabels(eventIngester *installv1alpha1.EventIngester) map[string]string {
	baseLabels := map[string]string{"release": eventIngester.Name}
	additionalLabels := getEventIngesterAdditionalLabels(eventIngester)
	baseLabels = MergeMaps(baseLabels, additionalLabels)
	identityLabels := getEventIngesterIdentityLabels(eventIngester)
	baseLabels = MergeMaps(baseLabels, identityLabels)
	return baseLabels
}

func getEventIngesterIdentityLabels(eventIngester *installv1alpha1.EventIngester) map[string]string {
	return map[string]string{"app": eventIngester.Name}
}

func getEventIngesterAdditionalLabels(eventIngester *installv1alpha1.EventIngester) map[string]string {
	m := make(map[string]string, len(eventIngester.Labels))
	for k, v := range eventIngester.Labels {
		m[k] = v
	}
	return m
}
