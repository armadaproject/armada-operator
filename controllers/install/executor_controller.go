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
	installv1alpha1 "github.com/dejanzele/armada-operator/apis/install/v1alpha1"
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
	"sigs.k8s.io/yaml"
)

var (
	executorFinalizer = fmt.Sprintf("executor.%s/finalizer", installv1alpha1.GroupVersion.Group)
)

// ExecutorReconciler reconciles a Executor object
type ExecutorReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=install.armadaproject.io,resources=executors,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=install.armadaproject.io,resources=executors/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=install.armadaproject.io,resources=executors/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Executor object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *ExecutorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var executor installv1alpha1.Executor
	if err := r.Client.Get(ctx, req.NamespacedName, &executor); err != nil {
		if k8serrors.IsNotFound(err) {
			logger.Info("Executor not found in cache, ending reconcile...", "namespace", req.Namespace, "name", req.Name)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	components, err := generateExecutorInstallComponents(&executor)
	if err != nil {
		return ctrl.Result{}, err
	}

	// THIS BLOCK MIGHT NOT BE NEEDED IF OWNER REFERENCES WORK
	// examine DeletionTimestamp to determine if object is under deletion
	if executor.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !controllerutil.ContainsFinalizer(&executor, executorFinalizer) {
			controllerutil.AddFinalizer(&executor, executorFinalizer)
			if err := r.Update(ctx, &executor); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// The object is being deleted
		if controllerutil.ContainsFinalizer(&executor, executorFinalizer) {
			// our finalizer is present, so lets handle any external dependency
			if err := r.deleteExternalResources(ctx, &executor, components); err != nil {
				// if fail to delete the external dependency here, return with error
				// so that it can be retried
				return ctrl.Result{}, err
			}

			// remove our finalizer from the list and update it.
			controllerutil.RemoveFinalizer(&executor, executorFinalizer)
			if err := r.Update(ctx, &executor); err != nil {
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
	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, components.Service, nil)
	if err != nil {
		return ctrl.Result{}, err
	}

	// now do init logic

	return ctrl.Result{}, nil
}

func doReconcile(mockK8sClient client) {
	mockK8sClient.EXPECT().Get(dsadf, fesjlgnse).Returns(fdsjlngd, fdsklnfsdl)
}

func (r *ExecutorReconciler) deleteExternalResources(ctx context.Context, executor *installv1alpha1.Executor, components *ExecutorComponents) error {
	return nil
}

type ExecutorComponents struct {
	Deployment         *appsv1.Deployment
	Service            *corev1.Service
	ServiceAccount     *corev1.ServiceAccount
	Secret             *corev1.Secret
	ClusterRole        *rbacv1.ClusterRole
	ClusterRoleBinding *rbacv1.ClusterRoleBinding
}

func generateExecutorInstallComponents(executor *installv1alpha1.Executor) (*ExecutorComponents, error) {
	owner := metav1.OwnerReference{
		APIVersion: executor.APIVersion,
		Kind:       executor.Kind,
		Name:       executor.Name,
		UID:        executor.UID,
	}
	ownerReference := []metav1.OwnerReference{owner}
	secret, err := createSecret(executor, ownerReference)
	if err != nil {
		return nil, err
	}
	deployment := createDeployment(executor, ownerReference)
	service := createService(executor, ownerReference)
	clusterRole := createClusterRole(executor, ownerReference)

	return &ExecutorComponents{
		Deployment:     deployment,
		Service:        service,
		ServiceAccount: nil,
		Secret:         secret,
		ClusterRole:    clusterRole,
	}, nil
}

func createSecret(executor *installv1alpha1.Executor, ownerReference []metav1.OwnerReference) (*corev1.Secret, error) {
	armadaConfig, err := generateArmadaConfig(nil)
	if err != nil {
		return nil, err
	}
	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: executor.Name, Namespace: executor.Namespace, OwnerReferences: ownerReference},
		Data:       armadaConfig,
	}
	return &secret, nil
}

func createDeployment(executor *installv1alpha1.Executor, ownerReference []metav1.OwnerReference) *appsv1.Deployment {
	var replicas int32 = 1
	deployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: executor.Name, Namespace: executor.Namespace, OwnerReferences: ownerReference},
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

func createService(executor *installv1alpha1.Executor, ownerReference []metav1.OwnerReference) *corev1.Service {
	service := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: executor.Name, Namespace: executor.Namespace, OwnerReferences: ownerReference},
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

func createClusterRole(executor *installv1alpha1.Executor, ownerReference []metav1.OwnerReference) *rbacv1.ClusterRole {
	podRules := rbacv1.PolicyRule{
		Verbs:     []string{"get", "list", "watch", "create", "delete", "deletecollection", "patch", "update"},
		APIGroups: []string{""},
		Resources: []string{"pods"},
	}
	eventRules := rbacv1.PolicyRule{
		Verbs:     []string{"get", "list", "watch", "delete", "deletecollection", "patch"},
		APIGroups: []string{""},
		Resources: []string{"events"},
	}
	serviceRules := rbacv1.PolicyRule{
		Verbs:     []string{"get", "list", "watch", "create", "delete", "deletecollection"},
		APIGroups: []string{""},
		Resources: []string{"services"},
	}
	nodeRules := rbacv1.PolicyRule{
		Verbs:     []string{"get", "list", "watch"},
		APIGroups: []string{""},
		Resources: []string{"nodes"},
	}
	nodeProxyRules := rbacv1.PolicyRule{
		Verbs:     []string{"get"},
		APIGroups: []string{""},
		Resources: []string{"node/proxy"},
	}
	userRules := rbacv1.PolicyRule{
		Verbs:     []string{"impersonate"},
		APIGroups: []string{""},
		Resources: []string{"users", "groups"},
	}
	ingressRules := rbacv1.PolicyRule{
		Verbs:     []string{"get", "list", "watch", "create", "delete", "deletecollection"},
		APIGroups: []string{"networking.k8s.io"},
		Resources: []string{"ingresses"},
	}
	tokenRules := rbacv1.PolicyRule{
		Verbs:     []string{"create"},
		APIGroups: []string{""},
		Resources: []string{"serviceaccounts/token"},
	}
	tokenReviewRules := rbacv1.PolicyRule{
		Verbs:     []string{"create"},
		APIGroups: []string{"authentication.k8s.io"},
		Resources: []string{"tokenreviews"},
	}
	clusterRole := rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: executor.Name, Namespace: executor.Namespace, OwnerReferences: ownerReference},
		Rules:      []rbacv1.PolicyRule{podRules, eventRules, serviceRules, nodeRules, nodeProxyRules, userRules, ingressRules, tokenRules, tokenReviewRules},
	}
	return &clusterRole
}

func createRoleBinding(executor *installv1alpha1.Executor, ownerReference []metav1.OwnerReference) *rbacv1.ClusterRoleBinding {
	clusterRoleBinding := rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: executor.Name, Namespace: executor.Namespace, OwnerReferences: ownerReference},
		Subjects:   []rbacv1.Subject{},
		RoleRef:    rbacv1.RoleRef{},
	}
	return &clusterRoleBinding
}

func generateArmadaConfig(config map[string]any) (map[string][]byte, error) {
	data, err := toYaml(config)
	if err != nil {
		return nil, err
	}
	return map[string][]byte{"armada-config.yaml": data}, nil
}

func toYaml(data map[string]any) ([]byte, error) {
	return yaml.Marshal(data)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ExecutorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&installv1alpha1.Executor{}).
		Complete(r)
}
