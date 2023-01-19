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

// TODO: Pretty sure all these constants need re-visiting.
const (
	lookoutIngesterApplicationConfigKey = "lookout-ingester-config.yaml"
	lookoutIngesterVolumeConfigKey      = "user-config"
	lookoutIngesterFinalizer            = "batch.tutorial.kubebuilder.io/finalizer"
)

// LookoutIngesterReconciler reconciles a LookoutIngester object
type LookoutIngesterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=install.armadaproject.io,resources=lookoutingesters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=install.armadaproject.io,resources=lookoutingesters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=install.armadaproject.io,resources=lookoutingesters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the LookoutIngester object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
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

	components, err := r.generateInstallComponents(&lookoutIngester)
	if err != nil {
		return ctrl.Result{}, err
	}

	deletionTimestamp := lookoutIngester.ObjectMeta.DeletionTimestamp
	// examine DeletionTimestamp to determine if object is under deletion
	if deletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !controllerutil.ContainsFinalizer(&lookoutIngester, lookoutIngesterFinalizer) {
			logger.Info("Attaching finalizer to LookoutIngester object", "finalizer", lookoutIngesterFinalizer)
			controllerutil.AddFinalizer(&lookoutIngester, lookoutIngesterFinalizer)
			if err := r.Update(ctx, &lookoutIngester); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		logger.Info("LookoutIngester object is being deleted", "finalizer", lookoutIngesterFinalizer)
		// The object is being deleted
		if controllerutil.ContainsFinalizer(&lookoutIngester, lookoutIngesterFinalizer) {
			// our finalizer is present, so lets handle any external dependency
			logger.Info("Running cleanup function for LookoutIngester object", "finalizer", lookoutIngesterFinalizer)
			if err := r.deleteExternalResources(ctx, components); err != nil {
				// if fail to delete the external dependency here, return with error
				// so that it can be retried
				return ctrl.Result{}, err
			}

			// remove our finalizer from the list and update it.
			logger.Info("Removing finalizer from LookoutIngester object", "finalizer", lookoutIngesterFinalizer)
			controllerutil.RemoveFinalizer(&lookoutIngester, lookoutIngesterFinalizer)
			if err := r.Update(ctx, &lookoutIngester); err != nil {
				return ctrl.Result{}, err
			}
		}

		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	mutateFn := func() error { return nil }

	logger.Info("Upserting LookoutIngester ServiceAccount object")
	if components.ServiceAccount != nil {
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.ServiceAccount, mutateFn); err != nil {
			return ctrl.Result{}, err
		}
	}

	logger.Info("Upserting LookoutIngester Secret object")
	if components.Secret != nil {
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.Secret, mutateFn); err != nil {
			return ctrl.Result{}, err
		}
	}

	logger.Info("Upserting LookoutIngester Deployment object")
	if components.Deployment != nil {
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.Deployment, mutateFn); err != nil {
			return ctrl.Result{}, err
		}
	}

	logger.Info("Successfully reconciled LookoutIngester object", "durationMilis", time.Since(started).Milliseconds())

	return ctrl.Result{}, nil
}

type LookoutIngesterComponents struct {
	Deployment     *appsv1.Deployment
	ServiceAccount *corev1.ServiceAccount
	Secret         *corev1.Secret
}

// SetupWithManager sets up the controller with the Manager.
func (r *LookoutIngesterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&installv1alpha1.LookoutIngester{}).
		Complete(r)
}

func (r *LookoutIngesterReconciler) generateInstallComponents(lookoutIngester *installv1alpha1.LookoutIngester) (*LookoutIngesterComponents, error) {
	secret, err := builders.CreateSecret(lookoutIngester.Spec.ApplicationConfig, lookoutIngester.Name, lookoutIngester.Namespace, GetConfigFilename(lookoutIngester.Name))
	if err != nil {
		return nil, err
	}
	if err := controllerutil.SetOwnerReference(lookoutIngester, secret, r.Scheme); err != nil {
		return nil, err
	}
	deployment := r.createDeployment(lookoutIngester)
	if err := controllerutil.SetOwnerReference(lookoutIngester, deployment, r.Scheme); err != nil {
		return nil, err
	}
	// namespace-scoped lookoutIngester cannot own cluster-scoped ClusterRole resource
	//if err := controllerutil.SetOwnerReference(lookoutIngester, clusterRole, r.Scheme); err != nil {
	//	return nil, err
	//}
	// namespace-scoped lookoutIngester cannot own cluster-scoped ClusterRole resource
	//if err := controllerutil.SetOwnerReference(lookoutIngester, clusterRoleBinding, r.Scheme); err != nil {
	//	return nil, err
	//}

	return &LookoutIngesterComponents{
		Deployment:     deployment,
		ServiceAccount: nil,
		Secret:         secret,
	}, nil
}

// TODO: Flesh this out for lookoutingester
func (r *LookoutIngesterReconciler) createDeployment(lookoutIngester *installv1alpha1.LookoutIngester) *appsv1.Deployment {
	var replicas int32 = 1
	deployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: lookoutIngester.Name, Namespace: lookoutIngester.Namespace},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels:      nil,
				MatchExpressions: nil,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{},
				Spec:       corev1.PodSpec{},
			},
			Strategy:                appsv1.DeploymentStrategy{},
			MinReadySeconds:         0,
			RevisionHistoryLimit:    nil,
			Paused:                  false,
			ProgressDeadlineSeconds: nil,
		},
	}
	return &deployment
}

func (r *LookoutIngesterReconciler) deleteExternalResources(ctx context.Context, components *LookoutIngesterComponents) error {
	// Nothing to do for now.
	return nil
}
