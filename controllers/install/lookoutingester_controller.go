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

// LookoutIngesterReconciler reconciles a LookoutIngester object
type LookoutIngesterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=install.armadaproject.io,resources=lookoutingesters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=install.armadaproject.io,resources=lookoutingesters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core,resources=secrets;serviceaccounts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *LookoutIngesterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("namespace", req.Namespace, "name", req.Name)
	started := time.Now()
	logger.Info("Reconciling LookoutIngester object")

	logger.Info("Fetching LookoutIngester object from cache")
	var lookoutIngester installv1alpha1.LookoutIngester
	if err := r.Client.Get(ctx, req.NamespacedName, &lookoutIngester); err != nil {
		if k8serrors.IsNotFound(err) {
			logger.Info("LookoutIngester not found in cache, ending reconcile")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	logger.Info(fmt.Sprintf("LookoutIngester Name %s", lookoutIngester.Name))
	err := lookoutIngester.Spec.BuildPortConfig()
	if err != nil {
		return ctrl.Result{}, err
	}
	components, err := r.generateInstallComponents(&lookoutIngester)
	if err != nil {
		return ctrl.Result{}, err
	}

	deletionTimestamp := lookoutIngester.ObjectMeta.DeletionTimestamp

	// examine DeletionTimestamp to determine if object is under deletion
	if !deletionTimestamp.IsZero() {
		logger.Info("LookoutIngester is being deleted")

		// FIXME: Seems like something actually has to happen here?

		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil

	}

	componentsCopy := components.DeepCopy()

	mutateFn := func() error {
		components.ReconcileComponents(componentsCopy)
		return nil
	}

	if components.Secret != nil {
		logger.Info("Upserting LookoutIngester Secret object")
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.Secret, mutateFn); err != nil {
			return ctrl.Result{}, err
		}
	}

	if components.ServiceAccount != nil {
		logger.Info("Upserting LookoutIngester ServiceAccount object")
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.ServiceAccount, mutateFn); err != nil {
			return ctrl.Result{}, err
		}
	}

	if components.Deployment != nil {
		logger.Info("Upserting LookoutIngester Deployment object")
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.Deployment, mutateFn); err != nil {
			return ctrl.Result{}, err
		}
	}

	logger.Info("Successfully reconciled LookoutIngester object", "durationMilis", time.Since(started).Milliseconds())

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *LookoutIngesterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&installv1alpha1.LookoutIngester{}).
		Complete(r)
}

func (r *LookoutIngesterReconciler) generateInstallComponents(lookoutIngester *installv1alpha1.LookoutIngester) (*CommonComponents, error) {
	secret, err := builders.CreateSecret(lookoutIngester.Spec.ApplicationConfig, lookoutIngester.Name, lookoutIngester.Namespace, GetConfigFilename(lookoutIngester.Name))
	if err != nil {
		return nil, err
	}
	if err := controllerutil.SetOwnerReference(lookoutIngester, secret, r.Scheme); err != nil {
		return nil, err
	}
	deployment := r.createDeployment(lookoutIngester, secret)
	if err := controllerutil.SetOwnerReference(lookoutIngester, deployment, r.Scheme); err != nil {
		return nil, err
	}
	serviceAccount := builders.CreateServiceAccount(lookoutIngester.Name, lookoutIngester.Namespace, AllLabels(lookoutIngester.Name, lookoutIngester.Labels), lookoutIngester.Spec.ServiceAccount)
	if err := controllerutil.SetOwnerReference(lookoutIngester, serviceAccount, r.Scheme); err != nil {
		return nil, err
	}

	return &CommonComponents{
		Deployment:     deployment,
		ServiceAccount: serviceAccount,
		Secret:         secret,
	}, nil
}

// TODO: Flesh this out for lookoutingester
func (r *LookoutIngesterReconciler) createDeployment(lookoutIngester *installv1alpha1.LookoutIngester, secret *corev1.Secret) *appsv1.Deployment {
	var replicas int32 = 1
	var runAsUser int64 = 1000
	var runAsGroup int64 = 2000
	allowPrivilegeEscalation := false

	env := createEnv(lookoutIngester.Spec.Environment)
	volumes := createVolumes(lookoutIngester.Name, lookoutIngester.Spec.AdditionalVolumes)
	volumeMounts := createVolumeMounts(GetConfigFilename(lookoutIngester.Name), lookoutIngester.Spec.AdditionalVolumeMounts)

	deployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: lookoutIngester.Name, Namespace: lookoutIngester.Namespace, Labels: AllLabels(lookoutIngester.Name, lookoutIngester.Labels)},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: IdentityLabel(lookoutIngester.Name),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:        lookoutIngester.Name,
					Namespace:   lookoutIngester.Namespace,
					Labels:      AllLabels(lookoutIngester.Name, lookoutIngester.Labels),
					Annotations: map[string]string{"checksum/config": GenerateChecksumConfig(lookoutIngester.Spec.ApplicationConfig.Raw)},
				},
				Spec: corev1.PodSpec{
					TerminationGracePeriodSeconds: lookoutIngester.Spec.TerminationGracePeriodSeconds,
					SecurityContext: &corev1.PodSecurityContext{
						RunAsUser:  &runAsUser,
						RunAsGroup: &runAsGroup,
					},
					Containers: []corev1.Container{{
						Name:            "lookoutingester",
						ImagePullPolicy: "IfNotPresent",
						Image:           ImageString(lookoutIngester.Spec.Image),
						Args:            []string{"--config", "/config/application_config.yaml"},
						// FIXME(Clif): Needs to change
						Ports: []corev1.ContainerPort{{
							Name:          "metrics",
							ContainerPort: lookoutIngester.Spec.PortConfig.MetricsPort,
							Protocol:      "TCP",
						}},
						Env:             env,
						VolumeMounts:    volumeMounts,
						SecurityContext: &corev1.SecurityContext{AllowPrivilegeEscalation: &allowPrivilegeEscalation},
					}},
					Tolerations: lookoutIngester.Spec.Tolerations,
					Volumes:     volumes,
				},
			},
		},
	}
	if lookoutIngester.Spec.Resources != nil {
		deployment.Spec.Template.Spec.Containers[0].Resources = *lookoutIngester.Spec.Resources
		deployment.Spec.Template.Spec.Containers[0].Env = addGoMemLimit(deployment.Spec.Template.Spec.Containers[0].Env, *lookoutIngester.Spec.Resources)
	}
	return &deployment
}
