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

	installv1alpha1 "github.com/armadaproject/armada-operator/apis/install/v1alpha1"
	"github.com/armadaproject/armada-operator/controllers/builders"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ArmadaServerReconciler reconciles a ArmadaServer object
type ArmadaServerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=install.armadaproject.io,resources=armadaservers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=install.armadaproject.io,resources=armadaservers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=install.armadaproject.io,resources=armadaservers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ArmadaServerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("namespace", req.Namespace, "name", req.Name)
	started := time.Now()
	logger.Info("Reconciling ArmadaServer object")

	logger.Info("Fetching ArmadaServer object from cache")
	var as installv1alpha1.ArmadaServer
	if err := r.Client.Get(ctx, req.NamespacedName, &as); err != nil {
		if k8serrors.IsNotFound(err) {
			logger.Info("ArmadaServer not found in cache, ending reconcile")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	components, err := generateArmadaServerInstallComponents(&as, r.Scheme)
	if err != nil {
		return ctrl.Result{}, err
	}

	deletionTimestamp := as.ObjectMeta.DeletionTimestamp
	// examine DeletionTimestamp to determine if object is under deletion
	if deletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !controllerutil.ContainsFinalizer(&as, operatorFinalizer) {
			logger.Info("Attaching finalizer to ArmadaServer object", "finalizer", operatorFinalizer)
			controllerutil.AddFinalizer(&as, operatorFinalizer)
			if err := r.Update(ctx, &as); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		logger.Info("ArmadaServer object is being deleted", "finalizer", operatorFinalizer)
		// The object is being deleted
		if controllerutil.ContainsFinalizer(&as, operatorFinalizer) {
			// our finalizer is present, so lets handle any external dependency
			logger.Info("Running cleanup function for ArmadaServer object", "finalizer", operatorFinalizer)

			// remove our finalizer from the list and update it.
			logger.Info("Removing finalizer from ArmadaServer object", "finalizer", operatorFinalizer)
			controllerutil.RemoveFinalizer(&as, operatorFinalizer)
			if err := r.Update(ctx, &as); err != nil {
				return ctrl.Result{}, err
			}
		}

		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	mutateFn := func() error { return nil }

	if components.Deployment != nil {
		logger.Info("Upserting ArmadaServer Deployment object")
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.Deployment, mutateFn); err != nil {
			return ctrl.Result{}, err
		}
	}

	if components.Ingress != nil {
		logger.Info("Upserting ArmadaServer GRPC Ingress object")
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.Ingress, mutateFn); err != nil {
			return ctrl.Result{}, err
		}
	}

	if components.IngressRest != nil {
		logger.Info("Upserting ArmadaServer IngressRest object")
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.IngressRest, mutateFn); err != nil {
			return ctrl.Result{}, err
		}
	}

	if components.Service != nil {
		logger.Info("Upserting ArmadaServer Service object")
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.Service, mutateFn); err != nil {
			return ctrl.Result{}, err
		}
	}

	if components.ServiceAccount != nil {
		logger.Info("Upserting ArmadaServer ServiceAccount object")
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.ServiceAccount, mutateFn); err != nil {
			return ctrl.Result{}, err
		}
	}

	if components.Secret != nil {
		logger.Info("Upserting ArmadaServer Secret object")
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.Secret, mutateFn); err != nil {
			return ctrl.Result{}, err
		}
	}

	if components.PodDisruptionBudget != nil {
		logger.Info("Upserting ArmadaServer PodDisruptionBudget object")
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.PodDisruptionBudget, mutateFn); err != nil {
			return ctrl.Result{}, err
		}
	}

	if components.PrometheusRule != nil {
		logger.Info("Upserting ArmadaServer PrometheusRule object")
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.PrometheusRule, mutateFn); err != nil {
			return ctrl.Result{}, err
		}
	}

	if components.ServiceMonitor != nil {
		logger.Info("Upserting ArmadaServer ServiceMonitor object")
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.ServiceMonitor, mutateFn); err != nil {
			return ctrl.Result{}, err
		}
	}

	logger.Info("Successfully reconciled ArmadaServer object", "durationMillis", time.Since(started).Milliseconds())

	return ctrl.Result{}, nil
}

type ArmadaServerComponents struct {
	Deployment          *appsv1.Deployment
	Ingress             *networkingv1.Ingress
	IngressRest         *networkingv1.Ingress
	Service             *corev1.Service
	ServiceAccount      *corev1.ServiceAccount
	Secret              *corev1.Secret
	PodDisruptionBudget *policyv1.PodDisruptionBudget
	PrometheusRule      *monitoringv1.PrometheusRule
	ServiceMonitor      *monitoringv1.ServiceMonitor
}

func generateArmadaServerInstallComponents(as *installv1alpha1.ArmadaServer, scheme *runtime.Scheme) (*ArmadaServerComponents, error) {
	deployment := createArmadaServerDeployment(as)
	if err := controllerutil.SetOwnerReference(as, deployment, scheme); err != nil {
		return nil, err
	}

	ingressGRPC := createIngressGRPC(as)
	if err := controllerutil.SetOwnerReference(as, ingressGRPC, scheme); err != nil {
		return nil, err
	}

	ingressRest := createIngressREST(as)
	if err := controllerutil.SetOwnerReference(as, ingressRest, scheme); err != nil {
		return nil, err
	}

	service := createArmadaServerService(as)
	if err := controllerutil.SetOwnerReference(as, service, scheme); err != nil {
		return nil, err
	}

	svcAcct := createArmadaServerServiceAccount(as)
	if err := controllerutil.SetOwnerReference(as, svcAcct, scheme); err != nil {
		return nil, err
	}

	secret, err := builders.CreateSecret(as.Spec.ApplicationConfig, as.Name, as.Namespace, GetConfigFilename(as.Name))
	if err != nil {
		return nil, err
	}
	if err := controllerutil.SetOwnerReference(as, secret, scheme); err != nil {
		return nil, err
	}

	pdb := createPodDisruptionBudget(as)
	if err := controllerutil.SetOwnerReference(as, pdb, scheme); err != nil {
		return nil, err
	}

	pr := createPrometheusRule(as)
	if err := controllerutil.SetOwnerReference(as, pr, scheme); err != nil {
		return nil, err
	}

	sm := createServiceMonitor(as)
	if err := controllerutil.SetOwnerReference(as, sm, scheme); err != nil {
		return nil, err
	}

	return &ArmadaServerComponents{
		Deployment:          deployment,
		Ingress:             ingressGRPC,
		IngressRest:         ingressRest,
		Service:             service,
		ServiceAccount:      svcAcct,
		Secret:              secret,
		PodDisruptionBudget: pdb,
		PrometheusRule:      pr,
		ServiceMonitor:      sm,
	}, nil

}

func createArmadaServerDeployment(as *installv1alpha1.ArmadaServer) *appsv1.Deployment {
	var replicas int32 = 1
	deployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: as.Name, Namespace: as.Namespace},
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

func createArmadaServerService(as *installv1alpha1.ArmadaServer) *corev1.Service {
	service := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: as.Name, Namespace: as.Namespace},
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

func createArmadaServerServiceAccount(as *installv1alpha1.ArmadaServer) *corev1.ServiceAccount {
	sa := corev1.ServiceAccount{
		ObjectMeta:                   metav1.ObjectMeta{Name: as.Name, Namespace: as.Namespace},
		Secrets:                      []corev1.ObjectReference{},
		ImagePullSecrets:             []corev1.LocalObjectReference{},
		AutomountServiceAccountToken: nil,
	}
	return &sa
}

func createIngressGRPC(as *installv1alpha1.ArmadaServer) *networkingv1.Ingress {
	grpcIngress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: as.Name, Namespace: as.Namespace, Labels: AllLabels(as.Name, as.Labels),
			Annotations: map[string]string{
				"kubernetes.io/ingress.class":                  as.Spec.Ingress.IngressClass,
				"nginx.ingress.kubernetes.io/ssl-redirect":     "true",
				"nginx.ingress.kubernetes.io/backend-protocol": "GRPC",
				"certmanager.k8s.io/cluster-issuer":            as.Spec.ClusterIssuer,
				"cert-manager.io/cluster-issuer":               as.Spec.ClusterIssuer,
			},
		},
	}
	if as.Spec.Ingress.Annotations != nil {
		for key, value := range as.Spec.Ingress.Annotations {
			grpcIngress.ObjectMeta.Annotations[key] = value
		}
	}
	if as.Spec.Ingress.Labels != nil {
		for key, value := range as.Spec.Ingress.Labels {
			grpcIngress.ObjectMeta.Labels[key] = value
		}
	}
	if as.Spec.Labels != nil {
		for key, value := range as.Spec.Labels {
			grpcIngress.ObjectMeta.Labels[key] = value
		}
	}
	if len(as.Spec.HostNames) > 0 {
		secretName := as.Name + "-service-tls"
		grpcIngress.Spec.TLS = []networking.IngressTLS{{Hosts: as.Spec.HostNames, SecretName: secretName}}
		ingressRules := []networking.IngressRule{}
		serviceName := "armada" + "-" + as.Name
		for _, val := range as.Spec.HostNames {
			ingressRules = append(ingressRules, networking.IngressRule{Host: val, IngressRuleValue: networking.IngressRuleValue{
				HTTP: &networking.HTTPIngressRuleValue{
					Paths: []networking.HTTPIngressPath{{
						Path:     "/",
						PathType: (*networking.PathType)(pointer.String("ImplementationSpecific")),
						Backend: networking.IngressBackend{
							Service: &networking.IngressServiceBackend{
								Name: serviceName,
								Port: networking.ServiceBackendPort{
									Number: 50051,
								},
							},
						},
					}},
				},
			}})
		}
		grpcIngress.Spec.Rules = ingressRules
	}

	return grpcIngress
}

func createIngressREST(as *installv1alpha1.ArmadaServer) *networkingv1.Ingress {
	restIngress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name: as.Name, Namespace: as.Namespace, Labels: AllLabels(as.Name, as.Labels),
			Annotations: map[string]string{
				"kubernetes.io/ingress.class":                as.Spec.Ingress.IngressClass,
				"certmanager.k8s.io/cluster-issuer":          as.Spec.ClusterIssuer,
				"cert-manager.io/cluster-issuer":             as.Spec.ClusterIssuer,
				"nginx.ingress.kubernetes.io/rewrite-target": "/$2",
				"nginx.ingress.kubernetes.io/ssl-redirect":   "true",
			},
		},
	}

	if as.Spec.Ingress.Annotations != nil {
		for key, value := range as.Spec.Ingress.Annotations {
			restIngress.ObjectMeta.Annotations[key] = value
		}
	}
	if as.Spec.Ingress.Labels != nil {
		for key, value := range as.Spec.Ingress.Labels {
			restIngress.ObjectMeta.Labels[key] = value
		}
	}
	if len(as.Spec.HostNames) > 0 {
		secretName := as.Name + "-service-tls"
		restIngress.Spec.TLS = []networking.IngressTLS{{Hosts: as.Spec.HostNames, SecretName: secretName}}
		ingressRules := []networking.IngressRule{}
		serviceName := "armada" + "-" + as.Name
		for _, val := range as.Spec.HostNames {
			ingressRules = append(ingressRules, networking.IngressRule{Host: val, IngressRuleValue: networking.IngressRuleValue{
				HTTP: &networking.HTTPIngressRuleValue{
					Paths: []networking.HTTPIngressPath{{
						Path:     "/api(/|$)(.*)",
						PathType: (*networking.PathType)(pointer.String("ImplementationSpecific")),
						Backend: networking.IngressBackend{
							Service: &networking.IngressServiceBackend{
								Name: serviceName,
								Port: networking.ServiceBackendPort{
									Number: 8080,
								},
							},
						},
					}},
				},
			}})
		}
		restIngress.Spec.Rules = ingressRules
	}
	return restIngress
}

func createPodDisruptionBudget(as *installv1alpha1.ArmadaServer) *policyv1.PodDisruptionBudget {
	return &policyv1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{Name: as.Name, Namespace: as.Namespace},
		Spec:       policyv1.PodDisruptionBudgetSpec{},
		Status:     policyv1.PodDisruptionBudgetStatus{},
	}
}

func createPrometheusRule(as *installv1alpha1.ArmadaServer) *monitoringv1.PrometheusRule {
	return &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{Name: as.Name, Namespace: as.Namespace},
		Spec: monitoringv1.PrometheusRuleSpec{
			Groups: []monitoringv1.RuleGroup{},
		},
	}
}

func createServiceMonitor(as *installv1alpha1.ArmadaServer) *monitoringv1.ServiceMonitor {
	return &monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{Name: as.Name, Namespace: as.Namespace},
		Spec:       monitoringv1.ServiceMonitorSpec{},
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *ArmadaServerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&installv1alpha1.ArmadaServer{}).
		Complete(r)
}
