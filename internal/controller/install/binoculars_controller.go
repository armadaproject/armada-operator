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

	"k8s.io/utils/ptr"

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

	logger.Info("Fetching Binoculars object from cache")

	var binoculars installv1alpha1.Binoculars
	if err := r.Client.Get(ctx, req.NamespacedName, &binoculars); err != nil {
		if k8serrors.IsNotFound(err) {
			logger.Info("Binoculars not found in cache, ending reconcile...", "namespace", req.Namespace, "name", req.Name)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	pc, err := installv1alpha1.BuildPortConfig(binoculars.Spec.ApplicationConfig)
	if err != nil {
		return ctrl.Result{}, err
	}
	binoculars.Spec.PortConfig = pc

	var components *CommonComponents
	components, err = generateBinocularsInstallComponents(&binoculars, r.Scheme)
	if err != nil {
		return ctrl.Result{}, err
	}

	deletionTimestamp := binoculars.ObjectMeta.DeletionTimestamp
	// examine DeletionTimestamp to determine if object is under deletion
	if deletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !controllerutil.ContainsFinalizer(&binoculars, operatorFinalizer) {
			logger.Info("Attaching finalizer to Binoculars object", "finalizer", operatorFinalizer)
			controllerutil.AddFinalizer(&binoculars, operatorFinalizer)
			if err := r.Update(ctx, &binoculars); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		logger.Info("Binoculars object is being deleted", "finalizer", operatorFinalizer)
		// The object is being deleted
		if controllerutil.ContainsFinalizer(&binoculars, operatorFinalizer) {
			// our finalizer is present, so lets handle any external dependency
			logger.Info("Running cleanup function for Binoculars object", "finalizer", operatorFinalizer)
			if err := r.deleteExternalResources(ctx, components); err != nil {
				// if fail to delete the external dependency here, return with error
				// so that it can be retried
				return ctrl.Result{}, err
			}

			// remove our finalizer from the list and update it.
			logger.Info("Removing finalizer from Binoculars object", "finalizer", operatorFinalizer)
			controllerutil.RemoveFinalizer(&binoculars, operatorFinalizer)
			if err := r.Update(ctx, &binoculars); err != nil {
				return ctrl.Result{}, err
			}
		}

		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	componentsCopy := components.DeepCopy()

	mutateFn := func() error {
		components.ReconcileComponents(componentsCopy)
		return nil
	}

	if components.ServiceAccount != nil {
		logger.Info("Upserting Binoculars ServiceAccount object")
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.ServiceAccount, mutateFn); err != nil {
			return ctrl.Result{}, err
		}
	}

	if components.ClusterRole != nil {
		logger.Info("Upserting Binoculars ClusterRole object")
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.ClusterRole, mutateFn); err != nil {
			return ctrl.Result{}, err
		}
	}

	if components.ClusterRoleBindings != nil && len(components.ClusterRoleBindings) > 0 {
		logger.Info("Upserting Binoculars ClusterRoleBinding object")
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.ClusterRoleBindings[0], mutateFn); err != nil {
			return ctrl.Result{}, err
		}
	}

	if components.Secret != nil {
		logger.Info("Upserting Binoculars Secret object")
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.Secret, mutateFn); err != nil {
			return ctrl.Result{}, err
		}
	}

	if components.Deployment != nil {
		logger.Info("Upserting Binoculars Deployment object")
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.Deployment, mutateFn); err != nil {
			return ctrl.Result{}, err
		}
	}

	if components.Service != nil {
		logger.Info("Upserting Binoculars Service object")
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.Service, mutateFn); err != nil {
			return ctrl.Result{}, err
		}
	}
	if components.IngressGrpc != nil {
		logger.Info("Upserting GRPC Ingress object")
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.IngressGrpc, mutateFn); err != nil {
			return ctrl.Result{}, err
		}
	}
	if components.IngressHttp != nil {
		logger.Info("Upserting REST Ingress object")
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.IngressHttp, mutateFn); err != nil {
			return ctrl.Result{}, err
		}
	}

	logger.Info("Successfully reconciled Binoculars object", "durationMillis", time.Since(started).Milliseconds())

	return ctrl.Result{}, nil
}

func generateBinocularsInstallComponents(binoculars *installv1alpha1.Binoculars, scheme *runtime.Scheme) (*CommonComponents, error) {
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
		serviceAccount = builders.CreateServiceAccount(binoculars.Name, binoculars.Namespace, AllLabels(binoculars.Name, binoculars.Labels), binoculars.Spec.ServiceAccount)
		if err = controllerutil.SetOwnerReference(binoculars, serviceAccount, scheme); err != nil {
			return nil, errors.WithStack(err)
		}
		serviceAccountName = serviceAccount.Name
	}
	deployment, err := createBinocularsDeployment(binoculars, secret, serviceAccountName)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if err = controllerutil.SetOwnerReference(binoculars, deployment, scheme); err != nil {
		return nil, errors.WithStack(err)
	}
	service := builders.Service(binoculars.Name, binoculars.Namespace, AllLabels(binoculars.Name, binoculars.Labels), IdentityLabel(binoculars.Name), binoculars.Spec.PortConfig)
	if err = controllerutil.SetOwnerReference(binoculars, service, scheme); err != nil {
		return nil, errors.WithStack(err)
	}

	ingress, err := createBinocularsIngressHttp(binoculars)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if err = controllerutil.SetOwnerReference(binoculars, ingress, scheme); err != nil {
		return nil, errors.WithStack(err)
	}

	ingressGrpc, err := createBinocularsIngressGrpc(binoculars)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if err = controllerutil.SetOwnerReference(binoculars, ingressGrpc, scheme); err != nil {
		return nil, errors.WithStack(err)
	}

	clusterRole := createBinocularsClusterRole(binoculars)
	clusterRoleBinding := generateBinocularsClusterRoleBinding(*binoculars)

	return &CommonComponents{
		Deployment:          deployment,
		Service:             service,
		ServiceAccount:      serviceAccount,
		Secret:              secret,
		ClusterRole:         clusterRole,
		ClusterRoleBindings: []*rbacv1.ClusterRoleBinding{clusterRoleBinding},
		IngressGrpc:         ingressGrpc,
		IngressHttp:         ingress,
	}, nil
}

// Function to build the deployment object for Binoculars.
// This should be changing from CRD to CRD.  Not sure if generalizing this helps much
func createBinocularsDeployment(binoculars *installv1alpha1.Binoculars, secret *corev1.Secret, serviceAccountName string) (*appsv1.Deployment, error) {
	env := createEnv(binoculars.Spec.Environment)
	volumes := createVolumes(binoculars.Name, binoculars.Spec.AdditionalVolumes)
	volumeMounts := createVolumeMounts(GetConfigFilename(secret.Name), binoculars.Spec.AdditionalVolumeMounts)
	ports := []corev1.ContainerPort{
		{
			Name:          "metrics",
			ContainerPort: binoculars.Spec.PortConfig.MetricsPort,
			Protocol:      "TCP",
		},
		{
			Name:          "http",
			ContainerPort: binoculars.Spec.PortConfig.HttpPort,
			Protocol:      "TCP",
		},
		{
			Name:          "grpc",
			ContainerPort: binoculars.Spec.PortConfig.GrpcPort,
			Protocol:      "TCP",
		},
	}
	containers := []corev1.Container{{
		Name:            "binoculars",
		ImagePullPolicy: "IfNotPresent",
		Image:           ImageString(binoculars.Spec.Image),
		Args:            []string{appConfigFlag, appConfigFilepath},
		Ports:           ports,
		Env:             env,
		VolumeMounts:    volumeMounts,
		SecurityContext: binoculars.Spec.SecurityContext,
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
					Affinity: &corev1.Affinity{
						PodAffinity: &corev1.PodAffinity{
							PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{{
								Weight: 100,
								PodAffinityTerm: corev1.PodAffinityTerm{
									TopologyKey: "kubernetes.io/hostname",
									LabelSelector: &metav1.LabelSelector{
										MatchExpressions: []metav1.LabelSelectorRequirement{{
											Key:      "app",
											Operator: metav1.LabelSelectorOpIn,
											Values:   []string{binoculars.Name},
										}},
									},
								},
							}},
						},
					},
					Containers: containers,
					Volumes:    volumes,
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

func generateBinocularsClusterRoleBinding(binoculars installv1alpha1.Binoculars) *rbacv1.ClusterRoleBinding {
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

func createBinocularsIngressGrpc(binoculars *installv1alpha1.Binoculars) (*networking.Ingress, error) {
	if len(binoculars.Spec.HostNames) == 0 {
		// when no hostnames provided, no ingress can be configured
		return nil, nil
	}

	grpcIngressName := binoculars.Name + "-grpc"

	grpcIngress := &networking.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: grpcIngressName, Namespace: binoculars.Namespace, Labels: AllLabels(binoculars.Name, binoculars.Labels),
			Annotations: map[string]string{
				"kubernetes.io/ingress.class":                  binoculars.Spec.Ingress.IngressClass,
				"nginx.ingress.kubernetes.io/ssl-redirect":     "true",
				"nginx.ingress.kubernetes.io/backend-protocol": "GRPC",
			},
		},
	}

	if binoculars.Spec.ClusterIssuer != "" {
		grpcIngress.ObjectMeta.Annotations["certmanager.k8s.io/cluster-issuer"] = binoculars.Spec.ClusterIssuer
		grpcIngress.ObjectMeta.Annotations["cert-manager.io/cluster-issuer"] = binoculars.Spec.ClusterIssuer
	}

	if binoculars.Spec.Ingress.Annotations != nil {
		for key, value := range binoculars.Spec.Ingress.Annotations {
			grpcIngress.ObjectMeta.Annotations[key] = value
		}
	}
	grpcIngress.ObjectMeta.Labels = AllLabels(binoculars.Name, binoculars.Spec.Labels, binoculars.Spec.Ingress.Labels)

	secretName := binoculars.Name + "-service-tls"
	grpcIngress.Spec.TLS = []networking.IngressTLS{{Hosts: binoculars.Spec.HostNames, SecretName: secretName}}
	var ingressRules []networking.IngressRule
	serviceName := "armada" + "-" + binoculars.Name
	for _, val := range binoculars.Spec.HostNames {
		ingressRules = append(ingressRules, networking.IngressRule{Host: val, IngressRuleValue: networking.IngressRuleValue{
			HTTP: &networking.HTTPIngressRuleValue{
				Paths: []networking.HTTPIngressPath{{
					Path:     "/",
					PathType: (*networking.PathType)(ptr.To[string]("ImplementationSpecific")),
					Backend: networking.IngressBackend{
						Service: &networking.IngressServiceBackend{
							Name: serviceName,
							Port: networking.ServiceBackendPort{
								Number: binoculars.Spec.PortConfig.GrpcPort,
							},
						},
					},
				}},
			},
		}})
	}
	grpcIngress.Spec.Rules = ingressRules

	return grpcIngress, nil
}

func createBinocularsIngressHttp(binoculars *installv1alpha1.Binoculars) (*networking.Ingress, error) {
	if len(binoculars.Spec.HostNames) == 0 {
		// when no hostnames provided, no ingress can be configured
		return nil, nil
	}
	restIngressName := binoculars.Name + "-rest"
	restIngress := &networking.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: restIngressName, Namespace: binoculars.Namespace, Labels: AllLabels(binoculars.Name, binoculars.Labels),
			Annotations: map[string]string{
				"kubernetes.io/ingress.class":                binoculars.Spec.Ingress.IngressClass,
				"nginx.ingress.kubernetes.io/rewrite-target": "/$2",
				"nginx.ingress.kubernetes.io/ssl-redirect":   "true",
			},
		},
	}

	if binoculars.Spec.ClusterIssuer != "" {
		restIngress.ObjectMeta.Annotations["certmanager.k8s.io/cluster-issuer"] = binoculars.Spec.ClusterIssuer
		restIngress.ObjectMeta.Annotations["cert-manager.io/cluster-issuer"] = binoculars.Spec.ClusterIssuer
	}

	if binoculars.Spec.Ingress.Annotations != nil {
		for key, value := range binoculars.Spec.Ingress.Annotations {
			restIngress.ObjectMeta.Annotations[key] = value
		}
	}
	restIngress.ObjectMeta.Labels = AllLabels(binoculars.Name, binoculars.Spec.Labels, binoculars.Spec.Ingress.Labels)

	secretName := binoculars.Name + "-service-tls"
	restIngress.Spec.TLS = []networking.IngressTLS{{Hosts: binoculars.Spec.HostNames, SecretName: secretName}}
	var ingressRules []networking.IngressRule
	serviceName := binoculars.Name
	for _, val := range binoculars.Spec.HostNames {
		ingressRules = append(ingressRules, networking.IngressRule{Host: val, IngressRuleValue: networking.IngressRuleValue{
			HTTP: &networking.HTTPIngressRuleValue{
				Paths: []networking.HTTPIngressPath{{
					Path:     "/api(/|$)(.*)",
					PathType: (*networking.PathType)(ptr.To[string]("ImplementationSpecific")),
					Backend: networking.IngressBackend{
						Service: &networking.IngressServiceBackend{
							Name: serviceName,
							Port: networking.ServiceBackendPort{
								Number: binoculars.Spec.PortConfig.HttpPort,
							},
						},
					},
				}},
			},
		}})
	}
	restIngress.Spec.Rules = ingressRules

	return restIngress, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BinocularsReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&installv1alpha1.Binoculars{}).
		Complete(r)
}
