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

	"github.com/pkg/errors"

	installv1alpha1 "github.com/armadaproject/armada-operator/apis/install/v1alpha1"
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

const (
	executorApplicationConfigKey = "armada-config.yaml"
	executorVolumeConfigKey      = "user-config"
	executorFinalizer            = "batch.tutorial.kubebuilder.io/finalizer"
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
	logger := log.FromContext(ctx).WithValues("namespace", req.Namespace, "name", req.Name)
	started := time.Now()
	logger.Info("Reconciling Executor object")

	logger.Info("Fetching Executor object from cache")
	var executor installv1alpha1.Executor
	if err := r.Client.Get(ctx, req.NamespacedName, &executor); err != nil {
		if k8serrors.IsNotFound(err) {
			logger.Info("Executor not found in cache, ending reconcile")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	components, err := generateExecutorInstallComponents(&executor, r.Scheme)
	if err != nil {
		return ctrl.Result{}, err
	}

	deletionTimestamp := executor.ObjectMeta.DeletionTimestamp
	// examine DeletionTimestamp to determine if object is under deletion
	if deletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !controllerutil.ContainsFinalizer(&executor, executorFinalizer) {
			logger.Info("Attaching finalizer to Executor object", "finalizer", executorFinalizer)
			controllerutil.AddFinalizer(&executor, executorFinalizer)
			if err := r.Update(ctx, &executor); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		logger.Info("Executor object is being deleted", "finalizer", executorFinalizer)
		// The object is being deleted
		if controllerutil.ContainsFinalizer(&executor, executorFinalizer) {
			// our finalizer is present, so lets handle any external dependency
			logger.Info("Running cleanup function for Executor object", "finalizer", executorFinalizer)
			if err := r.deleteExternalResources(ctx, components); err != nil {
				// if fail to delete the external dependency here, return with error
				// so that it can be retried
				return ctrl.Result{}, err
			}

			// remove our finalizer from the list and update it.
			logger.Info("Removing finalizer from Executor object", "finalizer", executorFinalizer)
			controllerutil.RemoveFinalizer(&executor, executorFinalizer)
			if err := r.Update(ctx, &executor); err != nil {
				return ctrl.Result{}, err
			}
		}

		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	mutateFn := func() error { return nil }

	if components.ServiceAccount != nil {
		logger.Info("Upserting Executor ServiceAccount object")
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.ServiceAccount, mutateFn); err != nil {
			return ctrl.Result{}, err
		}
	}
	if components.Secret != nil {
		logger.Info("Upserting Executor Secret object")
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.Secret, mutateFn); err != nil {
			return ctrl.Result{}, err
		}
	}

	if components.Deployment != nil {
		logger.Info("Upserting Executor Deployment object")
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.Deployment, mutateFn); err != nil {
			return ctrl.Result{}, err
		}
	}

	if components.Service != nil {
		logger.Info("Upserting Executor Service object")
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.Service, mutateFn); err != nil {
			return ctrl.Result{}, err
		}
	}

	logger.Info("Successfully reconciled Executor object", "durationMilis", time.Since(started).Milliseconds())

	return ctrl.Result{}, nil
}

type ExecutorComponents struct {
	Deployment         *appsv1.Deployment
	Service            *corev1.Service
	ServiceAccount     *corev1.ServiceAccount
	Secret             *corev1.Secret
	ClusterRole        *rbacv1.ClusterRole
	ClusterRoleBinding *rbacv1.ClusterRoleBinding
}

func generateExecutorInstallComponents(executor *installv1alpha1.Executor, scheme *runtime.Scheme) (*ExecutorComponents, error) {
	secret, err := createSecret(executor)
	if err != nil {
		return nil, err
	}
	if err := controllerutil.SetOwnerReference(executor, secret, scheme); err != nil {
		return nil, err
	}
	deployment := createDeployment(executor)
	if err := controllerutil.SetOwnerReference(executor, deployment, scheme); err != nil {
		return nil, err
	}
	service := createService(executor)
	if err := controllerutil.SetOwnerReference(executor, service, scheme); err != nil {
		return nil, err
	}
	serviceAccount := createServiceAccount(executor)
	if err := controllerutil.SetOwnerReference(executor, serviceAccount, scheme); err != nil {
		return nil, err
	}

	clusterRole := createClusterRole(executor)
	clusterRoleBinding := createClusterRoleBinding(executor, clusterRole, serviceAccount)

	return &ExecutorComponents{
		Deployment:         deployment,
		Service:            service,
		ServiceAccount:     serviceAccount,
		Secret:             secret,
		ClusterRoleBinding: clusterRoleBinding,
	}, nil
}

func createSecret(executor *installv1alpha1.Executor) (*corev1.Secret, error) {
	armadaConfig, err := generateArmadaConfig(executor.Spec.ApplicationConfig)
	if err != nil {
		return nil, err
	}
	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: executor.Name, Namespace: executor.Namespace},
		Data:       armadaConfig,
	}
	return &secret, nil
}

func generateArmadaConfig(config runtime.RawExtension) (map[string][]byte, error) {
	yamlConfig, err := yaml.JSONToYAML(config.Raw)
	if err != nil {
		return nil, err
	}

	return map[string][]byte{executorApplicationConfigKey: yamlConfig}, nil
}

func createDeployment(executor *installv1alpha1.Executor) *appsv1.Deployment {
	var replicas int32 = 1
	var runAsUser int64 = 1000
	var runAsGroup int64 = 2000
	allowPrivilegeEscalation := false
	deployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: executor.Name, Namespace: executor.Namespace, Labels: getAllExecutorLabels(executor)},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: getExecutorIdentityLabels(executor),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:        executor.Name,
					Namespace:   executor.Namespace,
					Labels:      getAllExecutorLabels(executor),
					Annotations: map[string]string{"checksum/config": getExecutorChecksumConfig(executor)},
				},
				Spec: corev1.PodSpec{
					TerminationGracePeriodSeconds: executor.Spec.TerminationGracePeriodSeconds,
					SecurityContext: &corev1.PodSecurityContext{
						RunAsUser:  &runAsUser,
						RunAsGroup: &runAsGroup,
					},
					Containers: []corev1.Container{{
						Name:            "executor",
						ImagePullPolicy: "IfNotPresent",
						Image:           ImageString(executor.Spec.Image),
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
								Name:      executorVolumeConfigKey,
								ReadOnly:  true,
								MountPath: "/config/application_config.yaml",
								SubPath:   getExecutorConfigFilename(executor),
							},
						},
						SecurityContext: &corev1.SecurityContext{AllowPrivilegeEscalation: &allowPrivilegeEscalation},
					}},
					NodeSelector: executor.Spec.NodeSelector,
					Tolerations:  executor.Spec.Tolerations,
					Volumes: []corev1.Volume{{
						Name: executorVolumeConfigKey,
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName: getExecutorConfigName(executor),
							},
						},
					}},
				},
			},
		},
	}
	if executor.Spec.Resources != nil {
		deployment.Spec.Template.Spec.Containers[0].Resources = *executor.Spec.Resources
	}
	return &deployment
}

func createService(executor *installv1alpha1.Executor) *corev1.Service {
	service := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: executor.Name, Namespace: executor.Namespace, Labels: getAllExecutorLabels(executor)},
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

func createClusterRole(executor *installv1alpha1.Executor) *rbacv1.ClusterRole {
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
		ObjectMeta: metav1.ObjectMeta{Name: executor.Name, Labels: getAllExecutorLabels(executor)},
		Rules:      []rbacv1.PolicyRule{podRules, eventRules, serviceRules, nodeRules, nodeProxyRules, userRules, ingressRules, tokenRules, tokenReviewRules},
	}
	return &clusterRole
}

func createServiceAccount(executor *installv1alpha1.Executor) *corev1.ServiceAccount {
	serviceAccount := corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: executor.Name, Namespace: executor.Namespace, Labels: getAllExecutorLabels(executor)},
	}
	if executor.Spec.ServiceAccount != nil {
		serviceAccount.AutomountServiceAccountToken = executor.Spec.ServiceAccount.AutomountServiceAccountToken
		serviceAccount.Secrets = executor.Spec.ServiceAccount.Secrets
		serviceAccount.ImagePullSecrets = executor.Spec.ServiceAccount.ImagePullSecrets
	}

	return &serviceAccount
}

func createClusterRoleBinding(
	executor *installv1alpha1.Executor,
	clusterRole *rbacv1.ClusterRole,
	serviceAccount *corev1.ServiceAccount,
) *rbacv1.ClusterRoleBinding {
	clusterRoleBinding := rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: executor.Name, Labels: getAllExecutorLabels(executor)},
		Subjects: []rbacv1.Subject{{
			Kind:      "ServiceAccount",
			Name:      serviceAccount.Name,
			Namespace: serviceAccount.Namespace,
		}},
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			APIGroup: "rbac.authorization.k8s.io",
			Name:     clusterRole.Name,
		},
	}
	return &clusterRoleBinding
}

func (r *ExecutorReconciler) deleteExternalResources(ctx context.Context, components *ExecutorComponents) error {
	if err := r.Delete(ctx, components.ClusterRole); err != nil && !k8serrors.IsNotFound(err) {
		return errors.Wrapf(err, "error deleting ClusterRole %s", components.ClusterRole.Name)
	}
	if err := r.Delete(ctx, components.ClusterRoleBinding); err != nil && !k8serrors.IsNotFound(err) {
		return errors.Wrapf(err, "error deleting ClusterRoleBinding %s", components.ClusterRoleBinding.Name)
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ExecutorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&installv1alpha1.Executor{}).
		Complete(r)
}
