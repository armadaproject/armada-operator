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

	"k8s.io/apimachinery/pkg/util/intstr"

	"k8s.io/utils/ptr"

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

	installv1alpha1 "github.com/armadaproject/armada-operator/api/install/v1alpha1"
	"github.com/armadaproject/armada-operator/internal/controller/builders"
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
	logger := log.FromContext(ctx).WithValues("namespace", req.Namespace, "name", req.Name)

	started := time.Now()

	logger.Info("Reconciling object")

	var executor installv1alpha1.Executor
	if miss, err := getObject(ctx, r.Client, &executor, req.NamespacedName, logger); err != nil || miss {
		return ctrl.Result{}, err
	}

	pc, err := installv1alpha1.BuildPortConfig(executor.Spec.ApplicationConfig)
	if err != nil {
		return ctrl.Result{}, err
	}
	executor.Spec.PortConfig = pc

	components, err := r.generateExecutorInstallComponents(&executor, r.Scheme)
	if err != nil {
		return ctrl.Result{}, err
	}

	cleanupF := func(ctx context.Context) error {
		return r.deleteExternalResources(ctx, components, logger)
	}
	finish, err := checkAndHandleObjectDeletion(ctx, r.Client, &executor, operatorFinalizer, cleanupF, logger)
	if err != nil || finish {
		return ctrl.Result{}, err
	}

	componentsCopy := components.DeepCopy()

	mutateFn := func() error {
		components.ReconcileComponents(componentsCopy)
		return nil
	}

	if err := upsertObjectIfNeeded(ctx, r.Client, components.ServiceAccount, executor.Kind, mutateFn, logger); err != nil {
		return ctrl.Result{}, err
	}

	if err := upsertObjectIfNeeded(ctx, r.Client, components.ClusterRole, executor.Kind, mutateFn, logger); err != nil {
		return ctrl.Result{}, err
	}

	for _, crb := range components.ClusterRoleBindings {
		if err := upsertObjectIfNeeded(ctx, r.Client, crb, executor.Kind, mutateFn, logger); err != nil {
			return ctrl.Result{}, err
		}
	}

	if err := upsertObjectIfNeeded(ctx, r.Client, components.Secret, executor.Kind, mutateFn, logger); err != nil {
		return ctrl.Result{}, err
	}

	if err := upsertObjectIfNeeded(ctx, r.Client, components.Deployment, executor.Kind, mutateFn, logger); err != nil {
		return ctrl.Result{}, err
	}

	if err := upsertObjectIfNeeded(ctx, r.Client, components.Service, executor.Kind, mutateFn, logger); err != nil {
		return ctrl.Result{}, err
	}

	for _, pc := range components.PriorityClasses {
		if err := upsertObjectIfNeeded(ctx, r.Client, pc, executor.Kind, mutateFn, logger); err != nil {
			return ctrl.Result{}, err
		}
	}

	if err := upsertObjectIfNeeded(ctx, r.Client, components.PrometheusRule, executor.Kind, mutateFn, logger); err != nil {
		return ctrl.Result{}, err
	}

	if err := upsertObjectIfNeeded(ctx, r.Client, components.ServiceMonitor, executor.Kind, mutateFn, logger); err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("Successfully reconciled object", "durationMillis", time.Since(started).Milliseconds())

	return ctrl.Result{}, nil
}

func (r *ExecutorReconciler) generateExecutorInstallComponents(executor *installv1alpha1.Executor, scheme *runtime.Scheme) (*CommonComponents, error) {
	secret, err := builders.CreateSecret(executor.Spec.ApplicationConfig, executor.Name, executor.Namespace, GetConfigFilename(executor.Name))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if err = controllerutil.SetOwnerReference(executor, secret, scheme); err != nil {
		return nil, errors.WithStack(err)
	}
	var serviceAccount *corev1.ServiceAccount
	serviceAccountName := executor.Spec.CustomServiceAccount
	if serviceAccountName == "" {
		serviceAccount = builders.CreateServiceAccount(executor.Name, executor.Namespace, AllLabels(executor.Name, executor.Labels), executor.Spec.ServiceAccount)
		if err = controllerutil.SetOwnerReference(executor, serviceAccount, scheme); err != nil {
			return nil, errors.WithStack(err)
		}
		serviceAccountName = serviceAccount.Name
	}
	deployment := r.createDeployment(executor, serviceAccountName)
	if err = controllerutil.SetOwnerReference(executor, deployment, scheme); err != nil {
		return nil, errors.WithStack(err)
	}
	service := builders.Service(executor.Name, executor.Namespace, AllLabels(executor.Name, executor.Labels), IdentityLabel(executor.Name), executor.Spec.PortConfig)
	if err = controllerutil.SetOwnerReference(executor, service, scheme); err != nil {
		return nil, errors.WithStack(err)
	}

	clusterRole := r.createClusterRole(executor)
	var clusterRoleBindings []*rbacv1.ClusterRoleBinding
	if serviceAccountName != "" {
		clusterRoleBindings = make([]*rbacv1.ClusterRoleBinding, 0, len(executor.Spec.AdditionalClusterRoleBindings)+1)
		clusterRoleBindings = append(clusterRoleBindings, r.createClusterRoleBinding(executor, clusterRole, serviceAccountName))
		clusterRoleBindings = append(clusterRoleBindings, r.createAdditionalClusterRoleBindings(executor, serviceAccountName)...)
	}

	components := &CommonComponents{
		Deployment:          deployment,
		Service:             service,
		ServiceAccount:      serviceAccount,
		Secret:              secret,
		ClusterRoleBindings: clusterRoleBindings,
		PriorityClasses:     executor.Spec.PriorityClasses,
		ClusterRole:         clusterRole,
	}

	if executor.Spec.Prometheus != nil && executor.Spec.Prometheus.Enabled {
		components.ServiceMonitor = r.createServiceMonitor(executor)
		if err = controllerutil.SetOwnerReference(executor, components.ServiceMonitor, scheme); err != nil {
			return nil, err
		}

		components.PrometheusRule = createExecutorPrometheusRule(executor)
		if err = controllerutil.SetOwnerReference(executor, components.PrometheusRule, scheme); err != nil {
			return nil, err
		}
	}

	return components, nil
}

func (r *ExecutorReconciler) createDeployment(executor *installv1alpha1.Executor, serviceAccountName string) *appsv1.Deployment {
	var replicas int32 = 1
	var runAsUser int64 = 1000
	var runAsGroup int64 = 2000
	volumes := createVolumes(executor.Name, executor.Spec.AdditionalVolumes)
	volumeMounts := createVolumeMounts(GetConfigFilename(executor.Name), executor.Spec.AdditionalVolumeMounts)

	allowPrivilegeEscalation := false
	ports := []corev1.ContainerPort{{
		Name:          "metrics",
		ContainerPort: executor.Spec.PortConfig.MetricsPort,
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
	containers := []corev1.Container{{
		Name:            "executor",
		ImagePullPolicy: "IfNotPresent",
		Image:           ImageString(executor.Spec.Image),
		Args:            []string{appConfigFlag, appConfigFilepath},
		Ports:           ports,
		Env:             env,
		VolumeMounts:    volumeMounts,
		SecurityContext: &corev1.SecurityContext{AllowPrivilegeEscalation: &allowPrivilegeEscalation},
	}}
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
		deployment.Spec.Template.Spec.Containers[0].Env = addGoMemLimit(deployment.Spec.Template.Spec.Containers[0].Env, *executor.Spec.Resources)
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
	interval := "15s"
	if executor.Spec.Prometheus.ScrapeInterval != nil {
		interval = duration.ShortHumanDuration(executor.Spec.Prometheus.ScrapeInterval.Duration)
	}
	return &monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      executor.Name,
			Namespace: executor.Namespace,
			Labels:    AllLabels(executor.Name, executor.Spec.Labels, executor.Spec.Prometheus.Labels),
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: selectorLabels,
			},
			AttachMetadata: &monitoringv1.AttachMetadata{
				Node: ptr.To(false),
			},
			Endpoints: []monitoringv1.Endpoint{{
				Port:     "metrics",
				Interval: monitoringv1.Duration(interval),
			}},
		},
	}
}

func (r *ExecutorReconciler) deleteExternalResources(ctx context.Context, components *CommonComponents, logger logr.Logger) error {
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

// createExecutorPrometheusRule will provide a prometheus monitoring rule for the name and scrapeInterval
func createExecutorPrometheusRule(executor *installv1alpha1.Executor) *monitoringv1.PrometheusRule {
	// Update the restRequestHistogram expression to align with Helm
	restRequestHistogram := `histogram_quantile(0.95, ` +
		`sum(rate(rest_client_request_duration_seconds_bucket{service="` + executor.Name + `"}[2m])) by (endpoint, verb, url, le))`
	logRate := "sum(rate(log_messages[2m])) by (level)"

	// Set the group name and duration string to match the Helm template
	scrapeInterval := &metav1.Duration{Duration: defaultPrometheusInterval}
	if interval := executor.Spec.Prometheus.ScrapeInterval; interval != nil {
		scrapeInterval = &metav1.Duration{Duration: interval.Duration}
	}
	durationString := duration.ShortHumanDuration(scrapeInterval.Duration)
	objectMetaName := "armada-" + executor.Name + "-metrics"

	return &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      executor.Name,
			Namespace: executor.Namespace,
			Labels:    AllLabels(executor.Name, executor.Labels, executor.Spec.Prometheus.Labels),
		},
		Spec: monitoringv1.PrometheusRuleSpec{
			Groups: []monitoringv1.RuleGroup{{
				Name:     objectMetaName,
				Interval: ptr.To(monitoringv1.Duration(durationString)),
				Rules: []monitoringv1.Rule{
					{
						Record: "armada:executor:rest:request:histogram95",
						Expr:   intstr.IntOrString{StrVal: restRequestHistogram},
					},
					{
						Record: "armada:executor:log:rate",
						Expr:   intstr.IntOrString{StrVal: logRate},
					},
				},
			}},
		},
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *ExecutorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&installv1alpha1.Executor{}).
		Complete(r)
}
