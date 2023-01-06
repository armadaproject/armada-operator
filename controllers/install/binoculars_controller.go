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

	installv1alpha1 "github.com/armadaproject/armada-operator/apis/install/v1alpha1"
	"github.com/armadaproject/armada-operator/controllers/builders"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// BinocularsReconciler reconciles a Binoculars object
type BinocularsReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=install.armadaproject.io,resources=binoculars,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=install.armadaproject.io,resources=binoculars/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=install.armadaproject.io,resources=binoculars/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Server object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *BinocularsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	logger := log.FromContext(ctx)

	var binoculars installv1alpha1.Binoculars
	if err := r.Client.Get(ctx, req.NamespacedName, &binoculars); err != nil {
		if k8serrors.IsNotFound(err) {
			logger.Info("Binoculars not found in cache, ending reconcile...", "namespace", req.Namespace, "name", req.Name)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	var components *BinocularsComponents
	components, err := generateBinocularsInstallComponents(&binoculars, r.Scheme)
	if err != nil {
		return ctrl.Result{}, err
	}
	mutateFn := func() error { return nil }

	if components.ServiceAccount != nil {
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.ServiceAccount, mutateFn); err != nil {
			return ctrl.Result{}, err
		}
	}

	if components.ClusterRole != nil {
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.ClusterRole, mutateFn); err != nil {
			return ctrl.Result{}, err
		}
	}

	if components.ClusterRoleBinding != nil {
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.ClusterRoleBinding, mutateFn); err != nil {
			return ctrl.Result{}, err
		}
	}

	if components.Secret != nil {
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.Secret, mutateFn); err != nil {
			return ctrl.Result{}, err
		}
	}

	if components.Deployment != nil {
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.Deployment, mutateFn); err != nil {
			return ctrl.Result{}, err
		}
	}

	if components.Service != nil {
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.Service, mutateFn); err != nil {
			return ctrl.Result{}, err
		}
	}

	// now do init logic

	return ctrl.Result{}, nil
}

type BinocularsComponents struct {
	ClusterRole        *rbacv1.ClusterRole
	ClusterRoleBinding *rbacv1.ClusterRoleBinding
	Ingress            *networking.Ingress
	IngressRest        *networking.Ingress
	Deployment         *appsv1.Deployment
	Service            *corev1.Service
	ServiceAccount     *corev1.ServiceAccount
	Secret             *corev1.Secret
}

func generateBinocularsInstallComponents(binoculars *installv1alpha1.Binoculars, scheme *runtime.Scheme) (*BinocularsComponents, error) {
	secret, err := builders.CreateSecret(binoculars.Spec.ApplicationConfig, binoculars.Name, binoculars.Namespace)
	if err != nil {
		return nil, err
	}
	if err := controllerutil.SetOwnerReference(binoculars, secret, scheme); err != nil {
		return nil, err
	}
	deployment := createBinocularsDeployment(binoculars)
	if err := controllerutil.SetOwnerReference(binoculars, deployment, scheme); err != nil {
		return nil, err
	}
	service := createBinocularsService(binoculars)
	if err := controllerutil.SetOwnerReference(binoculars, service, scheme); err != nil {
		return nil, err
	}
	clusterRole := createBinocularsClusterRole(binoculars)
	if err := controllerutil.SetOwnerReference(binoculars, clusterRole, scheme); err != nil {
		return nil, err
	}

	return &BinocularsComponents{
		Deployment:     deployment,
		Service:        service,
		ServiceAccount: nil,
		Secret:         secret,
		ClusterRole:    clusterRole,
	}, nil
}

// Function to build the deployment object for Binoculars.
// This should be changing from CRD to CRD.  Not sure if generailize this helps much
func createBinocularsDeployment(binoculars *installv1alpha1.Binoculars) *appsv1.Deployment {
	var replicas int32 = 1
	deployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: binoculars.Name, Namespace: binoculars.Namespace},
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

func createBinocularsService(binoculars *installv1alpha1.Binoculars) *corev1.Service {
	service := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: binoculars.Name, Namespace: binoculars.Namespace},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Name:     "metrics",
				Protocol: corev1.ProtocolTCP,
				Port:     9001,
			}},
		},
	}
	return &service
}

func createBinocularsClusterRole(binoculars *installv1alpha1.Binoculars) *rbacv1.ClusterRole {
	binocularRules := rbacv1.PolicyRule{
		Verbs:     []string{"impersonate"},
		APIGroups: []string{""},
		Resources: []string{"users", "groups"},
	}
	clusterRole := rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: binoculars.Name, Namespace: binoculars.Namespace},
		Rules:      []rbacv1.PolicyRule{binocularRules},
	}
	return &clusterRole

}

// SetupWithManager sets up the controller with the Manager.
func (r *BinocularsReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&installv1alpha1.Binoculars{}).
		Complete(r)
}
