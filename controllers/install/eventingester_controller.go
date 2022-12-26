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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	installv1alpha1 "github.com/armadaproject/armada-operator/apis/install/v1alpha1"
)

var (
	eventIngesterFinalizer = fmt.Sprintf("eventIngester.%s/finalizer", installv1alpha1.GroupVersion.Group)
)

// EventIngesterReconciler reconciles a EventIngester object
type EventIngesterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=install.armadaproject.io,resources=eventingesters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=install.armadaproject.io,resources=eventingesters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=install.armadaproject.io,resources=eventingesters/finalizers,verbs=update

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
	logger := log.FromContext(ctx)

	var eventIngester installv1alpha1.EventIngester
	if err := r.Client.Get(ctx, req.NamespacedName, &eventIngester); err != nil {
		if k8serrors.IsNotFound(err) {
			logger.Info("EventIngester not found in cache, ending reconcile...", "namespace", req.Namespace, "name", req.Name)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	components, err := r.generateComponents(&eventIngester)
	if err != nil {
		return ctrl.Result{}, err
	}

	// THIS BLOCK MIGHT NOT BE NEEDED IF OWNER REFERENCES WORK
	// examine DeletionTimestamp to determine if object is under deletion
	if eventIngester.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !controllerutil.ContainsFinalizer(&eventIngester, eventIngesterFinalizer) {
			controllerutil.AddFinalizer(&eventIngester, eventIngesterFinalizer)
			if err := r.Update(ctx, &eventIngester); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// The object is being deleted
		if controllerutil.ContainsFinalizer(&eventIngester, eventIngesterFinalizer) {
			// our finalizer is present, so lets handle any external dependency
			if err := r.deleteExternalResources(ctx, &eventIngester, components); err != nil {
				// if fail to delete the external dependency here, return with error
				// so that it can be retried
				return ctrl.Result{}, err
			}

			// remove our finalizer from the list and update it.
			controllerutil.RemoveFinalizer(&eventIngester, eventIngesterFinalizer)
			if err := r.Update(ctx, &eventIngester); err != nil {
				return ctrl.Result{}, err
			}
		}

		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}
	// END OF BLOCK

	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, components.ServiceAccount, nil)
	if err != nil {
		return ctrl.Result{}, err
	}
	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, components.ClusterRole, nil)
	if err != nil {
		return ctrl.Result{}, err
	}
	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, components.ClusterRoleBinding, nil)
	if err != nil {
		return ctrl.Result{}, err
	}
	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, components.Secret, nil)
	if err != nil {
		return ctrl.Result{}, err
	}
	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, components.Deployment, nil)
	if err != nil {
		return ctrl.Result{}, err
	}

	// now do init logic

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EventIngesterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&installv1alpha1.EventIngester{}).
		Complete(r)
}

func (r *EventIngesterReconciler) deleteExternalResources(ctx context.Context, eventIngester *installv1alpha1.EventIngester, components *EventIngesterComponents) error {
	return nil
}

type EventIngesterComponents struct {
	Deployment         *appsv1.Deployment
	ServiceAccount     *corev1.ServiceAccount
	Secret             *corev1.Secret
	ClusterRole        *rbacv1.ClusterRole
	ClusterRoleBinding *rbacv1.ClusterRoleBinding
}

func (r *EventIngesterReconciler) generateComponents(eventIngester *installv1alpha1.EventIngester) (*EventIngesterComponents, error) {
	owner := metav1.OwnerReference{
		APIVersion: eventIngester.APIVersion,
		Kind:       eventIngester.Kind,
		Name:       eventIngester.Name,
		UID:        eventIngester.UID,
	}
	ownerReference := []metav1.OwnerReference{owner}
	secret, err := r.createSecret(eventIngester, ownerReference)
	if err != nil {
		return nil, err
	}
	deployment := r.createDeployment(eventIngester, ownerReference)
	clusterRole := r.createClusterRole(eventIngester, ownerReference)

	return &EventIngesterComponents{
		Deployment:     deployment,
		ServiceAccount: nil,
		Secret:         secret,
		ClusterRole:    clusterRole,
	}, nil
}

func (r *EventIngesterReconciler) createSecret(eventIngester *installv1alpha1.EventIngester, ownerReference []metav1.OwnerReference) (*corev1.Secret, error) {
	armadaConfig, err := generateArmadaConfig(eventIngester.Spec.ApplicationConfig)
	if err != nil {
		return nil, err
	}
	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: eventIngester.Name, Namespace: eventIngester.Namespace, OwnerReferences: ownerReference},
		Data:       armadaConfig,
	}
	return &secret, nil
}

func (r *EventIngesterReconciler) createDeployment(eventIngester *installv1alpha1.EventIngester, ownerReference []metav1.OwnerReference) *appsv1.Deployment {
	var replicas int32 = 1
	deployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: eventIngester.Name, Namespace: eventIngester.Namespace, OwnerReferences: ownerReference},
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

func (r *EventIngesterReconciler) createClusterRole(eventIngester *installv1alpha1.EventIngester, ownerReference []metav1.OwnerReference) *rbacv1.ClusterRole {
	clusterRole := rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: eventIngester.Name, Namespace: eventIngester.Namespace, OwnerReferences: ownerReference},
		Rules:      policyRules(),
	}
	return &clusterRole
}

func (r *EventIngesterReconciler) createRoleBinding(eventIngester *installv1alpha1.EventIngester, ownerReference []metav1.OwnerReference) *rbacv1.ClusterRoleBinding {
	clusterRoleBinding := rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: eventIngester.Name, Namespace: eventIngester.Namespace, OwnerReferences: ownerReference},
		Subjects:   []rbacv1.Subject{},
		RoleRef:    rbacv1.RoleRef{},
	}
	return &clusterRoleBinding
}
