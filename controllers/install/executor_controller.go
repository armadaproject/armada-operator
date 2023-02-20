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

	schedulingv1 "k8s.io/api/scheduling/v1"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/util/duration"

	"github.com/pkg/errors"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
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
	"github.com/armadaproject/armada-operator/controllers/builders"
)

// ExecutorReconciler reconciles a Executor object
type ExecutorReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=install.armadaproject.io,resources=executors,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=install.armadaproject.io,resources=executors/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=install.armadaproject.io,resources=executors/finalizers,verbs=update
//+kubebuilder:rbac:groups="";apps;monitoring.coreos.com;rbac.authorization.k8s.io;scheduling.k8s.io,resources=services;serviceaccounts;clusterroles;clusterrolebindings;deployments;prometheusrules;servicemonitors,verbs=get;list;create;update;patch;delete
/// Executor ClusterRole RBAC
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;delete;deletecollection;patch;update
//+kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;delete;deletecollection;patch
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;delete;deletecollection
//+kubebuilder:rbac:groups=discovery.k8s.io,resources=endpointslices,verbs=get;list;watch
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;delete;deletecollection
//+kubebuilder:rbac:groups="",resources=nodes,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=users;groups,verbs=impersonate
//+kubebuilder:rbac:groups="",resources=nodes/proxy,verbs=get
//+kubebuilder:rbac:groups="",resources=serviceaccounts/token,verbs=create
//+kubebuilder:rbac:groups=authentication.k8s.io,resources=tokenreviews,verbs=create

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

	components, err := r.generateExecutorInstallComponents(&executor, r.Scheme)
	if err != nil {
		return ctrl.Result{}, err
	}

	componentsCopy := components.DeepCopy()

	deletionTimestamp := executor.ObjectMeta.DeletionTimestamp
	// examine DeletionTimestamp to determine if object is under deletion
	if deletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !controllerutil.ContainsFinalizer(&executor, operatorFinalizer) {
			logger.Info("Attaching finalizer to Executor object", "finalizer", operatorFinalizer)
			controllerutil.AddFinalizer(&executor, operatorFinalizer)
			if err := r.Update(ctx, &executor); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		logger.Info("Executor object is being deleted", "finalizer", operatorFinalizer)
		logger.Info("Namespace-scoped resources will be deleted by Kubernetes based on their OwnerReference")
		// The object is being deleted
		if controllerutil.ContainsFinalizer(&executor, operatorFinalizer) {
			// our finalizer is present, so lets handle any external dependency
			logger.Info("Running cleanup function for Executor cluster-scoped components", "finalizer", operatorFinalizer)
			if err := r.deleteExternalResources(ctx, components, logger); err != nil {
				// if fail to delete the external dependency here, return with error
				// so that it can be retried
				return ctrl.Result{}, err
			}

			// remove our finalizer from the list and update it.
			logger.Info("Removing finalizer from Executor object", "finalizer", operatorFinalizer)
			controllerutil.RemoveFinalizer(&executor, operatorFinalizer)
			if err := r.Update(ctx, &executor); err != nil {
				return ctrl.Result{}, err
			}
		}

		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	mutateFn := func() error {
		r.reconcileComponents(components, componentsCopy)
		return nil
	}

	if components.ServiceAccount != nil {
		logger.Info("Upserting Executor ServiceAccount object")
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.ServiceAccount, mutateFn); err != nil {
			return ctrl.Result{}, err
		}
	}

	if components.ClusterRole != nil {
		logger.Info("Upserting Executor ClusterRole object")
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.ClusterRole, mutateFn); err != nil {
			return ctrl.Result{}, err
		}
	}

	for _, crb := range components.ClusterRoleBindings {
		logger.Info("Upserting additional Executor ClusterRoleBinding object", "name", crb.Name)
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, crb, mutateFn); err != nil {
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

	for _, pc := range components.PriorityClasses {
		logger.Info("Upserting additional Executor PriorityClass object", "name", pc.Name)
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, pc, mutateFn); err != nil {
			return ctrl.Result{}, err
		}
	}

	if components.PrometheusRule != nil {
		logger.Info("Upserting Executor PrometheusRule object")
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.PrometheusRule, mutateFn); err != nil {
			return ctrl.Result{}, err
		}
	}

	if components.ServiceMonitor != nil {
		logger.Info("Upserting Executor ServiceMonitor object")
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.ServiceMonitor, mutateFn); err != nil {
			return ctrl.Result{}, err
		}
	}

	logger.Info("Successfully reconciled Executor object", "durationMillis", time.Since(started).Milliseconds())

	return ctrl.Result{}, nil
}

func (r *ExecutorReconciler) reconcileComponents(oldComponents, newComponents *ExecutorComponents) {
	oldComponents.Secret.Data = newComponents.Secret.Data
	oldComponents.Secret.Labels = newComponents.Secret.Labels
	oldComponents.Secret.Annotations = newComponents.Secret.Annotations
	oldComponents.Deployment.Spec = newComponents.Deployment.Spec
	oldComponents.Deployment.Labels = newComponents.Deployment.Labels
	oldComponents.Deployment.Annotations = newComponents.Deployment.Annotations
	oldComponents.Service.Spec = newComponents.Service.Spec
	oldComponents.Service.Labels = newComponents.Service.Labels
	oldComponents.Service.Annotations = newComponents.Service.Annotations
	oldComponents.ClusterRole.Rules = newComponents.ClusterRole.Rules
	oldComponents.ClusterRole.Labels = newComponents.ClusterRole.Labels
	oldComponents.ClusterRole.Annotations = newComponents.ClusterRole.Annotations
	for i := range oldComponents.ClusterRoleBindings {
		oldComponents.ClusterRoleBindings[i].RoleRef = newComponents.ClusterRoleBindings[i].RoleRef
		oldComponents.ClusterRoleBindings[i].Subjects = newComponents.ClusterRoleBindings[i].Subjects
		oldComponents.ClusterRoleBindings[i].Labels = newComponents.ClusterRoleBindings[i].Labels
		oldComponents.ClusterRoleBindings[i].Annotations = newComponents.ClusterRoleBindings[i].Annotations
	}
	for i := range oldComponents.PriorityClasses {
		oldComponents.PriorityClasses[i].PreemptionPolicy = newComponents.PriorityClasses[i].PreemptionPolicy
		oldComponents.PriorityClasses[i].Value = newComponents.PriorityClasses[i].Value
		oldComponents.PriorityClasses[i].Description = newComponents.PriorityClasses[i].Description
		oldComponents.PriorityClasses[i].GlobalDefault = newComponents.PriorityClasses[i].GlobalDefault
		oldComponents.PriorityClasses[i].Labels = newComponents.PriorityClasses[i].Labels
		oldComponents.PriorityClasses[i].Annotations = newComponents.PriorityClasses[i].Annotations
	}

}

func (r *ExecutorReconciler) generateExecutorInstallComponents(executor *installv1alpha1.Executor, scheme *runtime.Scheme) (*ExecutorComponents, error) {
	secret, err := builders.CreateSecret(executor.Spec.ApplicationConfig, executor.Name, executor.Namespace, GetConfigFilename(executor.Name))
	if err != nil {
		return nil, err
	}
	if err := controllerutil.SetOwnerReference(executor, secret, scheme); err != nil {
		return nil, err
	}
	serviceAccount := builders.CreateServiceAccount(executor.Name, executor.Namespace, AllLabels(executor.Name, executor.Labels), executor.Spec.ServiceAccount)
	if err := controllerutil.SetOwnerReference(executor, serviceAccount, scheme); err != nil {
		return nil, err
	}
	deployment := r.createDeployment(executor, secret, serviceAccount)
	if err := controllerutil.SetOwnerReference(executor, deployment, scheme); err != nil {
		return nil, err
	}
	service := builders.Service(executor.Name, executor.Namespace, AllLabels(executor.Name, executor.Labels))
	if err := controllerutil.SetOwnerReference(executor, service, scheme); err != nil {
		return nil, err
	}

	clusterRole := r.createClusterRole(executor)
	serviceAccountName := executor.Spec.CustomServiceAccount
	if serviceAccountName == "" {
		serviceAccountName = serviceAccount.Name
	}
	clusterRoleBindings := make([]*rbacv1.ClusterRoleBinding, 0, len(executor.Spec.AdditionalClusterRoleBindings)+1)
	clusterRoleBindings = append(clusterRoleBindings, r.createClusterRoleBinding(executor, clusterRole, serviceAccountName))
	clusterRoleBindings = append(clusterRoleBindings, r.createAdditionalClusterRoleBindings(executor, serviceAccountName)...)

	components := &ExecutorComponents{
		Deployment:          deployment,
		Service:             service,
		ServiceAccount:      serviceAccount,
		Secret:              secret,
		ClusterRoleBindings: clusterRoleBindings,
		PriorityClasses:     executor.Spec.PriorityClasses,
		ClusterRole:         clusterRole,
	}

	if executor.Spec.Prometheus != nil && executor.Spec.Prometheus.Enabled {
		serviceMonitor := r.createServiceMonitor(executor)
		if err := controllerutil.SetOwnerReference(executor, serviceMonitor, scheme); err != nil {
			return nil, err
		}
		components.ServiceMonitor = serviceMonitor

		pr := createPrometheusRule(executor.Name, executor.Namespace, executor.Spec.Prometheus.ScrapeInterval)
		components.PrometheusRule = pr
	}

	return components, nil
}

type ExecutorComponents struct {
	Deployment          *appsv1.Deployment
	Service             *corev1.Service
	ServiceAccount      *corev1.ServiceAccount
	Secret              *corev1.Secret
	ClusterRole         *rbacv1.ClusterRole
	ClusterRoleBindings []*rbacv1.ClusterRoleBinding
	PriorityClasses     []*schedulingv1.PriorityClass
	PrometheusRule      *monitoringv1.PrometheusRule
	ServiceMonitor      *monitoringv1.ServiceMonitor
}

func (ec *ExecutorComponents) DeepCopy() *ExecutorComponents {
	var clusterRoleBindings []*rbacv1.ClusterRoleBinding
	for _, crb := range ec.ClusterRoleBindings {
		clusterRoleBindings = append(clusterRoleBindings, crb.DeepCopy())
	}
	var priorityClasses []*schedulingv1.PriorityClass
	for _, pc := range ec.PriorityClasses {
		priorityClasses = append(priorityClasses, pc.DeepCopy())
	}
	cloned := &ExecutorComponents{
		Deployment:          ec.Deployment.DeepCopy(),
		Service:             ec.Service.DeepCopy(),
		ServiceAccount:      ec.ServiceAccount.DeepCopy(),
		Secret:              ec.Secret.DeepCopy(),
		ClusterRole:         ec.ClusterRole.DeepCopy(),
		ClusterRoleBindings: clusterRoleBindings,
		PriorityClasses:     priorityClasses,
		PrometheusRule:      ec.PrometheusRule.DeepCopy(),
		ServiceMonitor:      ec.ServiceMonitor.DeepCopy(),
	}
	if ec.PrometheusRule != nil {
		cloned.PrometheusRule = ec.PrometheusRule.DeepCopy()
	}
	if ec.ServiceMonitor != nil {
		cloned.Service = ec.Service.DeepCopy()
	}
	return cloned
}

func (r *ExecutorReconciler) createDeployment(executor *installv1alpha1.Executor, secret *corev1.Secret, serviceAccount *corev1.ServiceAccount) *appsv1.Deployment {
	var replicas int32 = 1
	var runAsUser int64 = 1000
	var runAsGroup int64 = 2000
	appConfigMount := "/config/application_config.yaml"
	allowPrivilegeEscalation := false
	ports := []corev1.ContainerPort{{
		Name:          "metrics",
		ContainerPort: 9001,
		Protocol:      "TCP",
	}}
	env := []corev1.EnvVar{
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
	}
	env = append(env, executor.Spec.Environment...)
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      volumeConfigKey,
			ReadOnly:  true,
			MountPath: appConfigMount,
			SubPath:   GetConfigFilename(executor.Name),
		},
	}
	volumeMounts = append(volumeMounts, executor.Spec.AdditionalVolumeMounts...)
	containers := []corev1.Container{{
		Name:            "executor",
		ImagePullPolicy: "IfNotPresent",
		Image:           ImageString(executor.Spec.Image),
		Args:            []string{"--config", appConfigMount},
		Ports:           ports,
		Env:             env,
		VolumeMounts:    volumeMounts,
		SecurityContext: &corev1.SecurityContext{AllowPrivilegeEscalation: &allowPrivilegeEscalation},
	}}
	volumes := []corev1.Volume{{
		Name: volumeConfigKey,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: secret.Name,
			},
		},
	}}
	volumes = append(volumes, executor.Spec.AdditionalVolumes...)
	serviceAccountName := executor.Spec.CustomServiceAccount
	if serviceAccountName == "" {
		serviceAccountName = serviceAccount.Name
	}
	deployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: executor.Name, Namespace: executor.Namespace, Labels: AllLabels(executor.Name, executor.Labels)},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: IdentityLabel(executor.Name),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:        executor.Name,
					Namespace:   executor.Namespace,
					Labels:      AllLabels(executor.Name, executor.Labels),
					Annotations: map[string]string{"checksum/config": GenerateChecksumConfig(executor.Spec.ApplicationConfig.Raw)},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName:            serviceAccountName,
					TerminationGracePeriodSeconds: executor.Spec.TerminationGracePeriodSeconds,
					SecurityContext: &corev1.PodSecurityContext{
						RunAsUser:  &runAsUser,
						RunAsGroup: &runAsGroup,
					},
					Containers:   containers,
					NodeSelector: executor.Spec.NodeSelector,
					Tolerations:  executor.Spec.Tolerations,
					Volumes:      volumes,
				},
			},
		},
	}
	if executor.Spec.Resources != nil {
		deployment.Spec.Template.Spec.Containers[0].Resources = *executor.Spec.Resources
	}
	return &deployment
}

func (r *ExecutorReconciler) createClusterRole(executor *installv1alpha1.Executor) *rbacv1.ClusterRole {
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
	endpointSliceRules := rbacv1.PolicyRule{
		Verbs:     []string{"get", "list", "watch"},
		APIGroups: []string{"discovery.k8s.io"},
		Resources: []string{"endpointslices"},
	}
	ingressRules := rbacv1.PolicyRule{
		Verbs:     []string{"get", "list", "watch", "create", "delete", "deletecollection"},
		APIGroups: []string{"networking.k8s.io"},
		Resources: []string{"ingresses"},
	}
	nodeRules := rbacv1.PolicyRule{
		Verbs:     []string{"get", "list", "watch"},
		APIGroups: []string{""},
		Resources: []string{"nodes"},
	}
	nodeProxyRules := rbacv1.PolicyRule{
		Verbs:     []string{"get"},
		APIGroups: []string{""},
		Resources: []string{"nodes/proxy"},
	}
	userRules := rbacv1.PolicyRule{
		Verbs:     []string{"impersonate"},
		APIGroups: []string{""},
		Resources: []string{"users", "groups"},
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
		ObjectMeta: metav1.ObjectMeta{Name: executor.Name, Labels: AllLabels(executor.Name, executor.Labels)},
		Rules:      []rbacv1.PolicyRule{podRules, eventRules, serviceRules, endpointSliceRules, nodeRules, nodeProxyRules, userRules, ingressRules, tokenRules, tokenReviewRules},
	}
	return &clusterRole
}

func (r *ExecutorReconciler) createClusterRoleBinding(
	executor *installv1alpha1.Executor,
	clusterRole *rbacv1.ClusterRole,
	serviceAccountName string,
) *rbacv1.ClusterRoleBinding {
	clusterRoleBinding := rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: executor.Name, Labels: AllLabels(executor.Name, executor.Labels)},
		Subjects: []rbacv1.Subject{{
			Kind:      "ServiceAccount",
			Name:      serviceAccountName,
			Namespace: executor.Namespace,
		}},
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			APIGroup: rbacv1.GroupName,
			Name:     clusterRole.Name,
		},
	}
	return &clusterRoleBinding
}

func (r *ExecutorReconciler) createAdditionalClusterRoleBindings(executor *installv1alpha1.Executor, serviceAccountName string) []*rbacv1.ClusterRoleBinding {
	var bindings []*rbacv1.ClusterRoleBinding
	for _, b := range executor.Spec.AdditionalClusterRoleBindings {
		name := fmt.Sprintf("%s-%s", executor.Name, b.NameSuffix)
		binding := rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{Name: name, Labels: AllLabels(executor.Name, executor.Labels)},
			Subjects: []rbacv1.Subject{{
				Kind:      "ServiceAccount",
				Name:      serviceAccountName,
				Namespace: executor.Namespace,
			}},
			RoleRef: rbacv1.RoleRef{
				Kind:     "ClusterRole",
				APIGroup: rbacv1.GroupName,
				Name:     b.ClusterRoleName,
			},
		}
		bindings = append(bindings, &binding)
	}
	return bindings
}

func (r *ExecutorReconciler) createServiceMonitor(executor *installv1alpha1.Executor) *monitoringv1.ServiceMonitor {
	selectorLabels := IdentityLabel(executor.Name)
	durationString := duration.ShortHumanDuration(executor.Spec.Prometheus.ScrapeInterval.Duration)
	return &monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      executor.Name,
			Namespace: executor.Namespace,
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: selectorLabels,
			},
			AttachMetadata: &monitoringv1.AttachMetadata{
				Node: false,
			},
			Endpoints: []monitoringv1.Endpoint{{
				Port:     "metrics",
				Interval: monitoringv1.Duration(durationString),
			}},
		},
	}
}

func (r *ExecutorReconciler) deleteExternalResources(ctx context.Context, components *ExecutorComponents, logger logr.Logger) error {
	if err := r.Delete(ctx, components.ClusterRole); err != nil && !k8serrors.IsNotFound(err) {
		return errors.Wrapf(err, "error deleting ClusterRole %s", components.ClusterRole.Name)
	}
	logger.Info("Successfully deleted Executor ClusterRole")

	if components.PrometheusRule != nil {
		if err := r.Delete(ctx, components.PrometheusRule); err != nil && !k8serrors.IsNotFound(err) {
			return errors.Wrapf(err, "error deleting PrometheusRule %s", components.PrometheusRule.Name)
		}
		logger.Info("Successfully deleted Executor PrometheusRule")
	}

	for _, crb := range components.ClusterRoleBindings {
		if err := r.Delete(ctx, crb); err != nil && !k8serrors.IsNotFound(err) {
			return errors.Wrapf(err, "error deleting ClusterRoleBinding %s", crb.Name)
		}
		logger.Info("Successfully deleted Executor ClusterRoleBinding", "name", crb.Name)
	}

	for _, pc := range components.PriorityClasses {
		if err := r.Delete(ctx, pc); err != nil && !k8serrors.IsNotFound(err) {
			return errors.Wrapf(err, "error deleting ClusterRoleBinding %s", pc.Name)
		}
		logger.Info("Successfully deleted Executor PriorityClass", "name", pc.Name)
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ExecutorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&installv1alpha1.Executor{}).
		Complete(r)
}
