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

	"k8s.io/utils/ptr"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"

	installv1alpha1 "github.com/armadaproject/armada-operator/api/install/v1alpha1"
	"github.com/armadaproject/armada-operator/internal/controller/builders"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/yaml"
)

// ArmadaServerReconciler reconciles a ArmadaServer object
type ArmadaServerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=install.armadaproject.io,resources=armadaservers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=install.armadaproject.io,resources=armadaservers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="batch",resources=jobs,verbs=get;list;watch;create;delete;deletecollection;patch;update
//+kubebuilder:rbac:groups=monitoring.coreos.com,resources=prometheusrules;servicemonitors,verbs=get;list;create;update;patch;delete

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
			logger.Info("ArmadaServer not found in cache, ending reconcile...", "namespace", req.Namespace, "name", req.Name)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	pc, err := installv1alpha1.BuildPortConfig(as.Spec.ApplicationConfig)
	if err != nil {
		return ctrl.Result{}, err
	}
	as.Spec.PortConfig = pc
	var components *CommonComponents
	components, err = generateArmadaServerInstallComponents(&as, r.Scheme)
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
			logger.Info("Attaching finalizer to As object", "finalizer", operatorFinalizer)
			controllerutil.AddFinalizer(&as, operatorFinalizer)
			if err := r.Update(ctx, &as); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		logger.Info("ArmadaServer object is being deleted", "finalizer", operatorFinalizer)
		logger.Info("Namespace-scoped resources will be deleted by Kubernetes based on their OwnerReference")
		// The object is being deleted
		if controllerutil.ContainsFinalizer(&as, operatorFinalizer) {
			// our finalizer is present, so lets handle any external dependency
			logger.Info("Running cleanup function for ArmadaServer cluster-scoped components", "finalizer", operatorFinalizer)
			if err := r.deleteExternalResources(ctx, components, logger); err != nil {
				// if fail to delete the external dependency here, return with error
				// so that it can be retried
				return ctrl.Result{}, err
			}

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

	componentsCopy := components.DeepCopy()

	mutateFn := func() error {
		components.ReconcileComponents(componentsCopy)
		return nil
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

	if as.Spec.PulsarInit {
		for idx := range components.Jobs {
			err = func() error {
				if components.Jobs[idx] != nil {
					if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.Jobs[idx], mutateFn); err != nil {
						return err
					}
					ctxTimeout, cancel := context.WithTimeout(ctx, migrationTimeout)
					defer cancel()

					err := waitForJob(ctxTimeout, r.Client, components.Jobs[idx], migrationPollSleep)
					if err != nil {
						return err
					}
				}
				return nil
			}()
			if err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	if components.Deployment != nil {
		logger.Info("Upserting ArmadaServer Deployment object")
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.Deployment, mutateFn); err != nil {
			return ctrl.Result{}, err
		}
	}

	if components.Service != nil {
		logger.Info("Upserting ArmadaServer Service object")
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.Service, mutateFn); err != nil {
			return ctrl.Result{}, err
		}
	}

	if components.IngressGrpc != nil {
		logger.Info("Upserting ArmadaServer GRPC Ingress object")
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.IngressGrpc, mutateFn); err != nil {
			return ctrl.Result{}, err
		}
	}

	if components.IngressHttp != nil {
		logger.Info("Upserting ArmadaServer IngressHttp object")
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, components.IngressHttp, mutateFn); err != nil {
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

func generateArmadaServerInstallComponents(as *installv1alpha1.ArmadaServer, scheme *runtime.Scheme) (*CommonComponents, error) {
	secret, err := builders.CreateSecret(as.Spec.ApplicationConfig, as.Name, as.Namespace, GetConfigFilename(as.Name))
	if err != nil {
		return nil, err
	}
	if err := controllerutil.SetOwnerReference(as, secret, scheme); err != nil {
		return nil, err
	}

	deployment, err := createArmadaServerDeployment(as)
	if err != nil {
		return nil, err
	}
	if err := controllerutil.SetOwnerReference(as, deployment, scheme); err != nil {
		return nil, err
	}

	ingressGrpc, err := createIngressGrpc(as)
	if err != nil {
		return nil, err
	}
	if err := controllerutil.SetOwnerReference(as, ingressGrpc, scheme); err != nil {
		return nil, err
	}

	ingressHttp, err := createIngressHttp(as)
	if err != nil {
		return nil, err
	}
	if err := controllerutil.SetOwnerReference(as, ingressHttp, scheme); err != nil {
		return nil, err
	}

	service := builders.Service(as.Name, as.Namespace, AllLabels(as.Name, as.Labels), IdentityLabel(as.Name), as.Spec.PortConfig)
	if err := controllerutil.SetOwnerReference(as, service, scheme); err != nil {
		return nil, err
	}

	svcAcct := builders.CreateServiceAccount(as.Name, as.Namespace, AllLabels(as.Name, as.Labels), as.Spec.ServiceAccount)
	if err := controllerutil.SetOwnerReference(as, svcAcct, scheme); err != nil {
		return nil, err
	}

	pdb := createPodDisruptionBudget(as)
	if err := controllerutil.SetOwnerReference(as, pdb, scheme); err != nil {
		return nil, err
	}

	var pr *monitoringv1.PrometheusRule
	if as.Spec.Prometheus != nil && as.Spec.Prometheus.Enabled {
		pr = createServerPrometheusRule(as.Name, as.Namespace, as.Spec.Prometheus.ScrapeInterval, as.Spec.Labels, as.Spec.Prometheus.Labels)
	}

	sm := createServiceMonitor(as)
	if err := controllerutil.SetOwnerReference(as, sm, scheme); err != nil {
		return nil, err
	}

	jobs := []*batchv1.Job{{}}
	if as.Spec.PulsarInit {
		jobs, err = createArmadaServerMigrationJobs(as)
		if err != nil {
			return nil, err
		}

		for _, job := range jobs {
			if err := controllerutil.SetOwnerReference(as, job, scheme); err != nil {
				return nil, err
			}
		}
	}
	return &CommonComponents{
		Deployment:          deployment,
		IngressGrpc:         ingressGrpc,
		IngressHttp:         ingressHttp,
		Service:             service,
		ServiceAccount:      svcAcct,
		Secret:              secret,
		PodDisruptionBudget: pdb,
		PrometheusRule:      pr,
		ServiceMonitor:      sm,
		Jobs:                jobs,
	}, nil

}

func createArmadaServerMigrationJobs(as *installv1alpha1.ArmadaServer) ([]*batchv1.Job, error) {
	runAsUser := int64(1000)
	runAsGroup := int64(2000)
	terminationGracePeriodSeconds := as.Spec.TerminationGracePeriodSeconds
	allowPrivilegeEscalation := false
	parallelism := int32(1)
	completions := int32(1)
	backoffLimit := int32(0)

	appConfig, err := builders.ConvertRawExtensionToYaml(as.Spec.ApplicationConfig)
	if err != nil {
		return []*batchv1.Job{}, err
	}
	var asConfig AppConfig
	err = yaml.Unmarshal([]byte(appConfig), &asConfig)
	if err != nil {
		return []*batchv1.Job{}, err
	}

	// First job is to poll/wait for Pulsar to be fully started
	pulsarWaitJob := batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "wait-for-pulsar",
			Namespace:   as.Namespace,
			Labels:      AllLabels(as.Name, as.Labels),
			Annotations: map[string]string{"checksum/config": GenerateChecksumConfig(as.Spec.ApplicationConfig.Raw)},
		},
		Spec: batchv1.JobSpec{
			Parallelism:  &parallelism,
			Completions:  &completions,
			BackoffLimit: &backoffLimit,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "wait-for-pulsar",
					Namespace: as.Namespace,
					Labels:    AllLabels(as.Name, as.Labels),
				},
				Spec: corev1.PodSpec{
					RestartPolicy:                 "Never",
					TerminationGracePeriodSeconds: terminationGracePeriodSeconds,
					SecurityContext: &corev1.PodSecurityContext{
						RunAsUser:  &runAsUser,
						RunAsGroup: &runAsGroup,
					},
					Containers: []corev1.Container{{
						Name:            "wait-for-pulsar",
						ImagePullPolicy: "IfNotPresent",
						Image:           "alpine:3.16",
						Args: []string{
							"/bin/sh",
							"-c",
							`echo "Waiting for Pulsar... ($PULSARHOST:$PULSARPORT)"
							while ! nc -z $PULSARHOST $PULSARPORT; do sleep 1; done
              echo "Pulsar started!"`,
						},
						Ports: []corev1.ContainerPort{{
							Name:          "metrics",
							ContainerPort: as.Spec.PortConfig.MetricsPort,
							Protocol:      "TCP",
						}},
						Env: []corev1.EnvVar{
							{Name: "PULSARHOST", Value: asConfig.Pulsar.ArmadaInit.BrokerHost},
							{Name: "PULSARPORT", Value: fmt.Sprintf("%d", asConfig.Pulsar.ArmadaInit.Port)},
						},
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      volumeConfigKey,
								ReadOnly:  true,
								MountPath: "/config/application_config.yaml",
								SubPath:   as.Name,
							},
						},
						SecurityContext: &corev1.SecurityContext{AllowPrivilegeEscalation: &allowPrivilegeEscalation},
					}},
					NodeSelector: as.Spec.NodeSelector,
					Tolerations:  as.Spec.Tolerations,
					Volumes: []corev1.Volume{{
						Name: volumeConfigKey,
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName: as.Name,
							},
						},
					}},
				},
			},
		},
	}

	// Second job is actually create namespaces/topics/partitions in Pulsar
	initPulsarJob := batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "init-pulsar",
			Namespace:   as.Namespace,
			Labels:      AllLabels(as.Name, as.Labels),
			Annotations: map[string]string{"checksum/config": GenerateChecksumConfig(as.Spec.ApplicationConfig.Raw)},
		},
		Spec: batchv1.JobSpec{
			Parallelism:  &parallelism,
			Completions:  &completions,
			BackoffLimit: &backoffLimit,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "init-pulsar",
					Namespace: as.Namespace,
					Labels:    AllLabels(as.Name, as.Labels),
				},
				Spec: corev1.PodSpec{
					RestartPolicy:                 "Never",
					TerminationGracePeriodSeconds: terminationGracePeriodSeconds,
					SecurityContext:               &corev1.PodSecurityContext{},
					Containers: []corev1.Container{{
						Name:            "init-pulsar",
						ImagePullPolicy: "IfNotPresent",
						Image:           fmt.Sprintf("%v:%v", asConfig.Pulsar.ArmadaInit.Image.Repository, asConfig.Pulsar.ArmadaInit.Image.Tag),
						Args: []string{
							"/bin/sh",
							"-c",
							`echo -e "Initializing pulsar $PULSARADMINURL"
              bin/pulsar-admin --admin-url $PULSARADMINURL tenants create armada
              bin/pulsar-admin --admin-url $PULSARADMINURL namespaces create armada/armada
              bin/pulsar-admin --admin-url $PULSARADMINURL topics delete-partitioned-topic persistent://armada/armada/events -f || true
              bin/pulsar-admin --admin-url $PULSARADMINURL topics create-partitioned-topic persistent://armada/armada/events -p 2

              # Disable topic auto-creation to ensure an error is thrown on using the wrong topic
              # (Pulsar automatically created the public tenant and default namespace).
              bin/pulsar-admin --admin-url $PULSARADMINURL namespaces set-auto-topic-creation public/default --disable
              bin/pulsar-admin --admin-url $PULSARADMINURL namespaces set-auto-topic-creation armada/armada --disable`,
						},
						Ports: []corev1.ContainerPort{{
							Name:          "metrics",
							ContainerPort: as.Spec.PortConfig.MetricsPort,
							Protocol:      "TCP",
						}},
						Env: []corev1.EnvVar{
							{
								Name: "PULSARADMINURL",
								Value: fmt.Sprintf("%s://%s:%d", asConfig.Pulsar.ArmadaInit.Protocol,
									asConfig.Pulsar.ArmadaInit.BrokerHost, asConfig.Pulsar.ArmadaInit.AdminPort),
							},
						},
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      volumeConfigKey,
								ReadOnly:  true,
								MountPath: "/config/application_config.yaml",
								SubPath:   as.Name,
							},
						},
						SecurityContext: &corev1.SecurityContext{AllowPrivilegeEscalation: &allowPrivilegeEscalation},
					}},
					NodeSelector: as.Spec.NodeSelector,
					Tolerations:  as.Spec.Tolerations,
					Volumes: []corev1.Volume{{
						Name: volumeConfigKey,
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName: as.Name,
							},
						},
					}},
				},
			},
		},
	}

	return []*batchv1.Job{&pulsarWaitJob, &initPulsarJob}, nil
}

func createArmadaServerDeployment(as *installv1alpha1.ArmadaServer) (*appsv1.Deployment, error) {
	var replicas int32 = 1
	var runAsUser int64 = 1000
	var runAsGroup int64 = 2000
	allowPrivilegeEscalation := false
	env := createEnv(as.Spec.Environment)
	pulsarConfig, err := ExtractPulsarConfig(as.Spec.ApplicationConfig)
	if err != nil {
		return nil, err
	}
	volumes := createVolumes(as.Name, as.Spec.AdditionalVolumes)
	volumes = append(volumes, createPulsarVolumes(pulsarConfig)...)
	volumeMounts := createVolumeMounts(GetConfigFilename(as.Name), as.Spec.AdditionalVolumeMounts)
	volumeMounts = append(volumeMounts, createPulsarVolumeMounts(pulsarConfig)...)

	deployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      as.Name,
			Namespace: as.Namespace,
			Labels:    AllLabels(as.Name, as.Labels),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: IdentityLabel(as.Name),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      as.Name,
					Namespace: as.Namespace,
					Labels:    AllLabels(as.Name, as.Labels),
					Annotations: map[string]string{
						"checksum/config": GenerateChecksumConfig(as.Spec.ApplicationConfig.Raw),
					},
				},
				Spec: corev1.PodSpec{
					TerminationGracePeriodSeconds: as.DeletionGracePeriodSeconds,
					SecurityContext: &corev1.PodSecurityContext{
						RunAsUser:  &runAsUser,
						RunAsGroup: &runAsGroup,
					},
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
											Values:   []string{as.Name},
										}},
									},
								},
							}},
						},
					},
					Containers: []corev1.Container{{
						Name:            "armadaserver",
						ImagePullPolicy: "IfNotPresent",
						Image:           ImageString(as.Spec.Image),
						Args:            []string{appConfigFlag, appConfigFilepath},
						Ports: []corev1.ContainerPort{
							{
								Name:          "metrics",
								ContainerPort: as.Spec.PortConfig.MetricsPort,
								Protocol:      "TCP",
							},
							{
								Name:          "grpc",
								ContainerPort: as.Spec.PortConfig.GrpcPort,
								Protocol:      "TCP",
							},
							{
								Name:          "http",
								ContainerPort: as.Spec.PortConfig.HttpPort,
								Protocol:      "TCP",
							},
						},
						Env:             env,
						VolumeMounts:    volumeMounts,
						SecurityContext: &corev1.SecurityContext{AllowPrivilegeEscalation: &allowPrivilegeEscalation},
					}},
					Volumes: volumes,
				},
			},
			Strategy:                appsv1.DeploymentStrategy{},
			MinReadySeconds:         0,
			RevisionHistoryLimit:    nil,
			Paused:                  false,
			ProgressDeadlineSeconds: nil,
		},
	}
	if as.Spec.Resources != nil {
		deployment.Spec.Template.Spec.Containers[0].Resources = *as.Spec.Resources
	}
	deployment.Spec.Template.Spec.Containers[0].Env = addGoMemLimit(deployment.Spec.Template.Spec.Containers[0].Env, *as.Spec.Resources)

	return &deployment, nil
}

func createIngressGrpc(as *installv1alpha1.ArmadaServer) (*networkingv1.Ingress, error) {
	if len(as.Spec.HostNames) == 0 {
		// if no hostnames, no ingress can be configured
		return nil, nil
	}
	ingressGRPCName := as.Name + "-grpc"
	grpcIngress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: ingressGRPCName, Namespace: as.Namespace, Labels: AllLabels(as.Name, as.Labels),
			Annotations: map[string]string{
				"kubernetes.io/ingress.class":                  as.Spec.Ingress.IngressClass,
				"nginx.ingress.kubernetes.io/ssl-redirect":     "true",
				"nginx.ingress.kubernetes.io/backend-protocol": "GRPC",
			},
		},
	}

	if as.Spec.ClusterIssuer != "" {
		grpcIngress.ObjectMeta.Annotations["certmanager.k8s.io/cluster-issuer"] = as.Spec.ClusterIssuer
		grpcIngress.ObjectMeta.Annotations["cert-manager.io/cluster-issuer"] = as.Spec.ClusterIssuer
	}

	if as.Spec.Ingress.Annotations != nil {
		for key, value := range as.Spec.Ingress.Annotations {
			grpcIngress.ObjectMeta.Annotations[key] = value
		}
	}
	grpcIngress.ObjectMeta.Labels = AllLabels(as.Name, as.Spec.Labels, as.Spec.Ingress.Labels)

	secretName := as.Name + "-service-tls"
	grpcIngress.Spec.TLS = []networking.IngressTLS{{Hosts: as.Spec.HostNames, SecretName: secretName}}
	var ingressRules []networking.IngressRule
	serviceName := as.Name
	for _, val := range as.Spec.HostNames {
		ingressRules = append(ingressRules, networking.IngressRule{Host: val, IngressRuleValue: networking.IngressRuleValue{
			HTTP: &networking.HTTPIngressRuleValue{
				Paths: []networking.HTTPIngressPath{{
					Path:     "/",
					PathType: (*networking.PathType)(ptr.To[string]("ImplementationSpecific")),
					Backend: networking.IngressBackend{
						Service: &networking.IngressServiceBackend{
							Name: serviceName,
							Port: networking.ServiceBackendPort{
								Number: as.Spec.PortConfig.GrpcPort,
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

func createIngressHttp(as *installv1alpha1.ArmadaServer) (*networkingv1.Ingress, error) {
	if len(as.Spec.HostNames) == 0 {
		// when no hostnames, no ingress can be configured
		return nil, nil
	}
	restIngressName := as.Name + "-rest"
	restIngress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name: restIngressName, Namespace: as.Namespace, Labels: AllLabels(as.Name, as.Labels),
			Annotations: map[string]string{
				"kubernetes.io/ingress.class":                as.Spec.Ingress.IngressClass,
				"nginx.ingress.kubernetes.io/rewrite-target": "/$2",
				"nginx.ingress.kubernetes.io/ssl-redirect":   "true",
			},
		},
	}

	if as.Spec.ClusterIssuer != "" {
		restIngress.ObjectMeta.Annotations["certmanager.k8s.io/cluster-issuer"] = as.Spec.ClusterIssuer
		restIngress.ObjectMeta.Annotations["cert-manager.io/cluster-issuer"] = as.Spec.ClusterIssuer
	}

	if as.Spec.Ingress.Annotations != nil {
		for key, value := range as.Spec.Ingress.Annotations {
			restIngress.ObjectMeta.Annotations[key] = value
		}
	}
	restIngress.ObjectMeta.Labels = AllLabels(as.Name, as.Spec.Labels, as.Spec.Ingress.Labels)

	secretName := as.Name + "-service-tls"
	restIngress.Spec.TLS = []networking.IngressTLS{{Hosts: as.Spec.HostNames, SecretName: secretName}}
	var ingressRules []networking.IngressRule
	serviceName := as.Name
	for _, val := range as.Spec.HostNames {
		ingressRules = append(ingressRules, networking.IngressRule{Host: val, IngressRuleValue: networking.IngressRuleValue{
			HTTP: &networking.HTTPIngressRuleValue{
				Paths: []networking.HTTPIngressPath{{
					Path:     "/api(/|$)(.*)",
					PathType: (*networking.PathType)(ptr.To[string]("ImplementationSpecific")),
					Backend: networking.IngressBackend{
						Service: &networking.IngressServiceBackend{
							Name: serviceName,
							Port: networking.ServiceBackendPort{
								Number: as.Spec.PortConfig.HttpPort,
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

func createPodDisruptionBudget(as *installv1alpha1.ArmadaServer) *policyv1.PodDisruptionBudget {
	return &policyv1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{Name: as.Name, Namespace: as.Namespace},
		Spec:       policyv1.PodDisruptionBudgetSpec{},
		Status:     policyv1.PodDisruptionBudgetStatus{},
	}
}

func createServiceMonitor(as *installv1alpha1.ArmadaServer) *monitoringv1.ServiceMonitor {
	var prometheusLabels map[string]string
	if as.Spec.Prometheus != nil {
		prometheusLabels = as.Spec.Prometheus.Labels
	}
	return &monitoringv1.ServiceMonitor{
		TypeMeta: metav1.TypeMeta{
			Kind: "ServiceMonitor",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      as.Name,
			Namespace: as.Namespace,
			Labels:    AllLabels(as.Name, as.Spec.Labels, prometheusLabels),
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			Endpoints: []monitoringv1.Endpoint{
				{Port: "metrics", Interval: "15s"},
			},
		},
	}
}

// deleteExternalResources removes any external resources during deletion
func (r *ArmadaServerReconciler) deleteExternalResources(ctx context.Context, components *CommonComponents, logger logr.Logger) error {

	if components.PrometheusRule != nil {
		if err := r.Delete(ctx, components.PrometheusRule); err != nil && !k8serrors.IsNotFound(err) {
			return errors.Wrapf(err, "error deleting PrometheusRule %s", components.PrometheusRule.Name)
		}
		logger.Info("Successfully deleted ArmadaServer PrometheusRule")
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ArmadaServerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&installv1alpha1.ArmadaServer{}).
		Complete(r)
}

// createServerPrometheusRule will provide a prometheus monitoring rule for the name and scrapeInterval
func createServerPrometheusRule(name, namespace string, scrapeInterval *metav1.Duration, labels ...map[string]string) *monitoringv1.PrometheusRule {
	if scrapeInterval == nil {
		scrapeInterval = &metav1.Duration{Duration: defaultPrometheusInterval}
	}
	queueSize := `avg(sum(armada_queue_size) by (queueName, pod)) by (queueName) > 0`
	queuePriority := `avg(sum(armada_queue_priority) by (pool, queueName, pod)) by (pool, queueName)`
	queueIdeal := `(sum(armada:queue:resource:queued{resourceType="cpu"} > bool 0) by (queueName, pool) * (1 / armada:queue:priority))
			       / ignoring(queueName) group_left
		       sum(sum(armada:queue:resource:queued{resourceType="cpu"} > bool 0) by (queueName, pool) * (1 / armada:queue:priority)) by (pool)
		       * 100`
	queueResourceQueued := `avg(armada_queue_resource_queued) by (pool, queueName, resourceType)`
	queueResourceAllocated := `avg(armada_queue_resource_allocated) by (pool, cluster, queueName, resourceType, nodeType)`
	queueResourceUsed := `avg(armada_queue_resource_used) by (pool, cluster, queueName, resourceType, nodeType)`
	serverHist := `histogram_quantile(0.95, sum(rate(grpc_server_handling_seconds_bucket{grpc_type!="server_stream"}[2m])) by (grpc_method,grpc_service, le))`
	serverRequestRate := `sum(rate(grpc_server_handled_total[2m])) by (grpc_method,grpc_service)`
	logRate := `sum(rate(log_messages[2m])) by (level)`
	availableCapacity := `avg(armada_cluster_available_capacity) by (pool, cluster, resourceType, nodeType)`
	resourceCapacity := `avg(armada_cluster_capacity) by (pool, cluster, resourceType, nodeType)`
	queuePodPhaseCount := `max(armada_queue_leased_pod_count) by (pool, cluster, queueName, phase, nodeType)`

	durationString := duration.ShortHumanDuration(scrapeInterval.Duration)
	objectMetaName := "armada-" + name + "-metrics"
	return &monitoringv1.PrometheusRule{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    AllLabels(name, labels...),
		},
		Spec: monitoringv1.PrometheusRuleSpec{
			Groups: []monitoringv1.RuleGroup{{
				Name:     objectMetaName,
				Interval: monitoringv1.Duration(durationString),
				Rules: []monitoringv1.Rule{
					{
						Record: "armada:queue:size",
						Expr:   intstr.IntOrString{StrVal: queueSize},
					},
					{
						Record: "armada:queue:priority",
						Expr:   intstr.IntOrString{StrVal: queuePriority},
					},
					{
						Record: "armada:queue:ideal_current_share",
						Expr:   intstr.IntOrString{StrVal: queueIdeal},
					},
					{
						Record: "armada:queue:resource:queued",
						Expr:   intstr.IntOrString{StrVal: queueResourceQueued},
					},
					{
						Record: "armada:queue:resource:allocated",
						Expr:   intstr.IntOrString{StrVal: queueResourceAllocated},
					},
					{
						Record: "armada:queue:resource:used",
						Expr:   intstr.IntOrString{StrVal: queueResourceUsed},
					},
					{
						Record: "armada:grpc:server:histogram95",
						Expr:   intstr.IntOrString{StrVal: serverHist},
					},
					{
						Record: "armada:grpc:server:requestrate",
						Expr:   intstr.IntOrString{StrVal: serverRequestRate},
					},
					{
						Record: "armada:log:rate",
						Expr:   intstr.IntOrString{StrVal: logRate},
					},
					{
						Record: "armada:resource:available_capacity",
						Expr:   intstr.IntOrString{StrVal: availableCapacity},
					},
					{
						Record: "armada:resource:capacity",
						Expr:   intstr.IntOrString{StrVal: resourceCapacity},
					},
					{
						Record: "armada:queue:pod_phase:count",
						Expr:   intstr.IntOrString{StrVal: queuePodPhaseCount},
					},
				},
			}},
		},
	}
}
