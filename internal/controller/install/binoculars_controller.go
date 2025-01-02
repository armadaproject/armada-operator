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

	"github.com/armadaproject/armada-operator/internal/controller/common"

	"github.com/pkg/errors"

	installv1alpha1 "github.com/armadaproject/armada-operator/api/install/v1alpha1"
	"github.com/armadaproject/armada-operator/internal/controller/builders"

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
//+kubebuilder:rbac:groups="",resources=groups;users,verbs=impersonate
//+kubebuilder:rbac:groups=core,resources=secrets;services;serviceaccounts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles;clusterrolebindings;users,verbs=get;list;watch;create;update;patch;delete;impersonate

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *BinocularsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("namespace", req.Namespace, "name", req.Name)

	started := time.Now()

	logger.Info("Reconciling Binoculars object")

	var binoculars installv1alpha1.Binoculars
	if miss, err := common.GetObject(ctx, r.Client, &binoculars, req.NamespacedName, logger); err != nil || miss {
		return ctrl.Result{}, err
	}

	commonConfig, err := builders.ParseCommonApplicationConfig(binoculars.Spec.ApplicationConfig)
	if err != nil {
		return ctrl.Result{}, err
	}

	var components *CommonComponents
	components, err = generateBinocularsInstallComponents(&binoculars, r.Scheme, commonConfig)
	if err != nil {
		return ctrl.Result{}, err
	}

	cleanupF := func(ctx context.Context) error {
		return r.deleteExternalResources(ctx, components)
	}
	finish, err := checkAndHandleObjectDeletion(ctx, r.Client, &binoculars, operatorFinalizer, cleanupF, logger)
	if err != nil || finish {
		return ctrl.Result{}, err
	}

	componentsCopy := components.DeepCopy()

	mutateFn := func() error {
		components.ReconcileComponents(componentsCopy)
		return nil
	}

	if err := upsertObjectIfNeeded(ctx, r.Client, components.ServiceAccount, binoculars.Kind, mutateFn, logger); err != nil {
		return ctrl.Result{}, err
	}

	if err := upsertObjectIfNeeded(ctx, r.Client, components.ClusterRole, binoculars.Kind, mutateFn, logger); err != nil {
		return ctrl.Result{}, err
	}

	if len(components.ClusterRoleBindings) > 0 {
		for _, crb := range components.ClusterRoleBindings {
			if err := upsertObjectIfNeeded(ctx, r.Client, crb, binoculars.Kind, mutateFn, logger); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	if err := upsertObjectIfNeeded(ctx, r.Client, components.Secret, binoculars.Kind, mutateFn, logger); err != nil {
		return ctrl.Result{}, err
	}

	if err := upsertObjectIfNeeded(ctx, r.Client, components.Deployment, binoculars.Kind, mutateFn, logger); err != nil {
		return ctrl.Result{}, err
	}

	if err := upsertObjectIfNeeded(ctx, r.Client, components.Service, binoculars.Kind, mutateFn, logger); err != nil {
		return ctrl.Result{}, err
	}

	if err := upsertObjectIfNeeded(ctx, r.Client, components.IngressGrpc, binoculars.Kind, mutateFn, logger); err != nil {
		return ctrl.Result{}, err
	}

	if err := upsertObjectIfNeeded(ctx, r.Client, components.IngressHttp, binoculars.Kind, mutateFn, logger); err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("Successfully reconciled Binoculars object", "durationMillis", time.Since(started).Milliseconds())

	return ctrl.Result{}, nil
}

func generateBinocularsInstallComponents(binoculars *installv1alpha1.Binoculars, scheme *runtime.Scheme, config *builders.CommonApplicationConfig) (*CommonComponents, error) {
	secret, err := builders.CreateSecret(binoculars.Spec.ApplicationConfig, binoculars.Name, binoculars.Namespace, GetConfigFilename(binoculars.Name))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if err = controllerutil.SetOwnerReference(binoculars, secret, scheme); err != nil {
		return nil, errors.WithStack(err)
	}
	var serviceAccount *corev1.ServiceAccount
	serviceAccountName := binoculars.Spec.CustomServiceAccount
	if serviceAccountName == "" {
		serviceAccount = builders.ServiceAccount(
			binoculars.Name, binoculars.Namespace, AllLabels(binoculars.Name, binoculars.Labels), binoculars.Spec.ServiceAccount,
		)
		if err = controllerutil.SetOwnerReference(binoculars, serviceAccount, scheme); err != nil {
			return nil, errors.WithStack(err)
		}
		serviceAccountName = serviceAccount.Name
	}
	deployment, err := createBinocularsDeployment(binoculars, secret, serviceAccountName, config)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if err = controllerutil.SetOwnerReference(binoculars, deployment, scheme); err != nil {
		return nil, errors.WithStack(err)
	}
	service := builders.Service(
		binoculars.Name,
		binoculars.Namespace,
		AllLabels(binoculars.Name, binoculars.Labels),
		IdentityLabel(binoculars.Name),
		config,
		builders.ServiceEnableApplicationPortsOnly,
	)
	if err = controllerutil.SetOwnerReference(binoculars, service, scheme); err != nil {
		return nil, errors.WithStack(err)
	}
	profilingService, ingressProfiling, err := newProfilingComponents(
		binoculars,
		scheme,
		config,
		binoculars.Spec.ProfilingIngressConfig,
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	ingressHTTP, err := createBinocularsIngressHttp(binoculars, config)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if err = controllerutil.SetOwnerReference(binoculars, ingressHTTP, scheme); err != nil {
		return nil, errors.WithStack(err)
	}

	ingressGRPC, err := createBinocularsIngressGrpc(binoculars, config)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if err = controllerutil.SetOwnerReference(binoculars, ingressGRPC, scheme); err != nil {
		return nil, errors.WithStack(err)
	}

	clusterRole := createBinocularsClusterRole(binoculars)
	clusterRoleBinding := generateBinocularsClusterRoleBinding(binoculars)

	return &CommonComponents{
		Deployment:          deployment,
		Service:             service,
		ServiceProfiling:    profilingService,
		ServiceAccount:      serviceAccount,
		Secret:              secret,
		ClusterRole:         clusterRole,
		ClusterRoleBindings: []*rbacv1.ClusterRoleBinding{clusterRoleBinding},
		IngressGrpc:         ingressGRPC,
		IngressHttp:         ingressHTTP,
		IngressProfiling:    ingressProfiling,
	}, nil
}

// Function to build the deployment object for Binoculars.
// This should be changing from CRD to CRD.  Not sure if generalizing this helps much
func createBinocularsDeployment(
	binoculars *installv1alpha1.Binoculars,
	secret *corev1.Secret,
	serviceAccountName string,
	commonConfig *builders.CommonApplicationConfig,
) (*appsv1.Deployment, error) {
	env := createEnv(binoculars.Spec.Environment)
	volumes := createVolumes(binoculars.Name, binoculars.Spec.AdditionalVolumes)
	volumeMounts := createVolumeMounts(GetConfigFilename(secret.Name), binoculars.Spec.AdditionalVolumeMounts)
	readinessProbe, livenessProbe := CreateProbesWithScheme(GetServerScheme(commonConfig.GRPC.TLS))

	containers := []corev1.Container{{
		Name:            "binoculars",
		ImagePullPolicy: corev1.PullIfNotPresent,
		Image:           ImageString(binoculars.Spec.Image),
		Args:            []string{appConfigFlag, appConfigFilepath},
		Ports:           newContainerPortsAll(commonConfig),
		Env:             env,
		VolumeMounts:    volumeMounts,
		SecurityContext: binoculars.Spec.SecurityContext,
		ReadinessProbe:  readinessProbe,
		LivenessProbe:   livenessProbe,
	}}
	deployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: binoculars.Name, Namespace: binoculars.Namespace, Labels: AllLabels(binoculars.Name, binoculars.Labels)},
		Spec: appsv1.DeploymentSpec{
			Replicas: binoculars.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: IdentityLabel(binoculars.Name),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:        binoculars.Name,
					Namespace:   binoculars.Namespace,
					Labels:      AllLabels(binoculars.Name, binoculars.Labels),
					Annotations: map[string]string{"checksum/config": GenerateChecksumConfig(binoculars.Spec.ApplicationConfig.Raw)},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName:            serviceAccountName,
					TerminationGracePeriodSeconds: binoculars.DeletionGracePeriodSeconds,
					SecurityContext:               binoculars.Spec.PodSecurityContext,
					Affinity:                      defaultAffinity(binoculars.Name, 100),
					Containers:                    containers,
					Volumes:                       volumes,
				},
			},
		},
	}
	if binoculars.Spec.Resources != nil {
		deployment.Spec.Template.Spec.Containers[0].Resources = *binoculars.Spec.Resources
	}
	deployment.Spec.Template.Spec.Containers[0].Env = addGoMemLimit(deployment.Spec.Template.Spec.Containers[0].Env, *binoculars.Spec.Resources)

	return &deployment, nil
}

func createBinocularsClusterRole(binoculars *installv1alpha1.Binoculars) *rbacv1.ClusterRole {
	binocularRules := []rbacv1.PolicyRule{
		{
			Verbs:     []string{"impersonate"},
			APIGroups: []string{""},
			Resources: []string{"users", "groups"},
		},
		{
			Verbs:     []string{"get", "list", "watch", "patch"},
			APIGroups: []string{""},
			Resources: []string{"nodes"},
		},
	}
	clusterRole := rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: binoculars.Name},
		Rules:      binocularRules,
	}
	return &clusterRole
}

func generateBinocularsClusterRoleBinding(binoculars *installv1alpha1.Binoculars) *rbacv1.ClusterRoleBinding {
	clusterRoleBinding := rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   binoculars.Name,
			Labels: AllLabels(binoculars.Name, binoculars.Labels),
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     binoculars.Name,
		},
		Subjects: []rbacv1.Subject{{
			Kind:      "ServiceAccount",
			Name:      binoculars.Name,
			Namespace: binoculars.Namespace,
		},
		},
	}
	return &clusterRoleBinding
}
func (r *BinocularsReconciler) deleteExternalResources(ctx context.Context, components *CommonComponents) error {
	if err := r.Delete(ctx, components.ClusterRole); err != nil && !k8serrors.IsNotFound(err) {
		return errors.Wrapf(err, "error deleting ClusterRole %s", components.ClusterRole.Name)
	}
	if err := r.Delete(ctx, components.ClusterRoleBindings[0]); err != nil && !k8serrors.IsNotFound(err) {
		return errors.Wrapf(err, "error deleting ClusterRoleBinding %s", components.ClusterRoleBindings[0].Name)
	}
	return nil
}

func createBinocularsIngressGrpc(binoculars *installv1alpha1.Binoculars, config *builders.CommonApplicationConfig) (*networking.Ingress, error) {
	if len(binoculars.Spec.HostNames) == 0 {
		// when no hostnames, no ingress can be configured
		return nil, nil
	}
	name := binoculars.Name + "-grpc"
	labels := AllLabels(binoculars.Name, binoculars.Spec.Labels, binoculars.Spec.Ingress.Labels)
	baseAnnotations := map[string]string{
		"nginx.ingress.kubernetes.io/ssl-redirect": "true",
	}
	annotations := buildIngressAnnotations(binoculars.Spec.Ingress, baseAnnotations, BackendProtocolGRPC, config.GRPC.Enabled)

	secretName := binoculars.Name + "-service-tls"
	serviceName := binoculars.Name
	servicePort := config.HTTPPort
	path := "/"
	ingress, err := builders.Ingress(name, binoculars.Namespace, labels, annotations, binoculars.Spec.HostNames, serviceName, secretName, path, servicePort)
	return ingress, errors.WithStack(err)
}

func createBinocularsIngressHttp(binoculars *installv1alpha1.Binoculars, config *builders.CommonApplicationConfig) (*networking.Ingress, error) {
	if len(binoculars.Spec.HostNames) == 0 {
		// when no hostnames, no ingress can be configured
		return nil, nil
	}
	name := binoculars.Name + "-rest"
	labels := AllLabels(binoculars.Name, binoculars.Spec.Labels, binoculars.Spec.Ingress.Labels)
	baseAnnotations := map[string]string{
		"nginx.ingress.kubernetes.io/rewrite-target": "/$2",
		"nginx.ingress.kubernetes.io/ssl-redirect":   "true",
	}
	annotations := buildIngressAnnotations(binoculars.Spec.Ingress, baseAnnotations, BackendProtocolHTTP, config.GRPC.Enabled)

	secretName := binoculars.Name + "-service-tls"
	serviceName := binoculars.Name
	servicePort := config.HTTPPort
	path := "/api(/|$)(.*)"
	ingress, err := builders.Ingress(name, binoculars.Namespace, labels, annotations, binoculars.Spec.HostNames, serviceName, secretName, path, servicePort)
	return ingress, errors.WithStack(err)
}

// SetupWithManager sets up the controller with the Manager.
func (r *BinocularsReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&installv1alpha1.Binoculars{}).
		Complete(r)
}
