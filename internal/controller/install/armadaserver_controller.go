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

	"github.com/armadaproject/armada-operator/internal/controller/common"

	"k8s.io/utils/ptr"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"

	installv1alpha1 "github.com/armadaproject/armada-operator/api/install/v1alpha1"
	"github.com/armadaproject/armada-operator/internal/controller/builders"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
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

	logger.Info("Reconciling object")

	var server installv1alpha1.ArmadaServer
	if miss, err := common.GetObject(ctx, r.Client, &server, req.NamespacedName, logger); err != nil || miss {
		return ctrl.Result{}, err
	}

	commonConfig, err := builders.ParseCommonApplicationConfig(server.Spec.ApplicationConfig)
	if err != nil {
		return ctrl.Result{}, err
	}

	var components *CommonComponents
	components, err = generateArmadaServerInstallComponents(&server, r.Scheme, commonConfig)
	if err != nil {
		return ctrl.Result{}, err
	}

	cleanupF := func(ctx context.Context) error {
		return r.deleteExternalResources(ctx, components, logger)
	}
	finish, err := checkAndHandleObjectDeletion(ctx, r.Client, &server, operatorFinalizer, cleanupF, logger)
	if err != nil || finish {
		return ctrl.Result{}, err
	}

	componentsCopy := components.DeepCopy()

	mutateFn := func() error {
		components.ReconcileComponents(componentsCopy)
		return nil
	}

	if err := upsertObjectIfNeeded(ctx, r.Client, components.ServiceAccount, server.Kind, mutateFn, logger); err != nil {
		return ctrl.Result{}, err
	}

	if err := upsertObjectIfNeeded(ctx, r.Client, components.Secret, server.Kind, mutateFn, logger); err != nil {
		return ctrl.Result{}, err
	}

	if server.Spec.PulsarInit {
		for _, job := range components.Jobs {
			err = func(job *batchv1.Job) error {
				if err := upsertObjectIfNeeded(ctx, r.Client, job, server.Kind, mutateFn, logger); err != nil {
					return err
				}

				if err := waitForJob(ctx, r.Client, job, jobPollInterval, jobTimeout); err != nil {
					return err
				}
				return nil
			}(job)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	if err := upsertObjectIfNeeded(ctx, r.Client, components.Deployment, server.Kind, mutateFn, logger); err != nil {
		return ctrl.Result{}, err
	}

	if err := upsertObjectIfNeeded(ctx, r.Client, components.Service, server.Kind, mutateFn, logger); err != nil {
		return ctrl.Result{}, err
	}

	if err := upsertObjectIfNeeded(ctx, r.Client, components.IngressGrpc, server.Kind, mutateFn, logger); err != nil {
		return ctrl.Result{}, err
	}

	if err := upsertObjectIfNeeded(ctx, r.Client, components.IngressHttp, server.Kind, mutateFn, logger); err != nil {
		return ctrl.Result{}, err
	}

	if err := upsertObjectIfNeeded(ctx, r.Client, components.PodDisruptionBudget, server.Kind, mutateFn, logger); err != nil {
		return ctrl.Result{}, err
	}

	if err := upsertObjectIfNeeded(ctx, r.Client, components.PrometheusRule, server.Kind, mutateFn, logger); err != nil {
		return ctrl.Result{}, err
	}

	if err := upsertObjectIfNeeded(ctx, r.Client, components.ServiceMonitor, server.Kind, mutateFn, logger); err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("Successfully reconciled ArmadaServer object", "durationMillis", time.Since(started).Milliseconds())

	return ctrl.Result{}, nil
}

func generateArmadaServerInstallComponents(as *installv1alpha1.ArmadaServer, scheme *runtime.Scheme, config *builders.CommonApplicationConfig) (*CommonComponents, error) {
	secret, err := builders.CreateSecret(as.Spec.ApplicationConfig, as.Name, as.Namespace, GetConfigFilename(as.Name))
	if err != nil {
		return nil, err
	}
	if err := controllerutil.SetOwnerReference(as, secret, scheme); err != nil {
		return nil, err
	}

	var serviceAccount *corev1.ServiceAccount
	serviceAccountName := as.Spec.CustomServiceAccount
	if serviceAccountName == "" {
		serviceAccount = builders.ServiceAccount(as.Name, as.Namespace, AllLabels(as.Name, as.Labels), as.Spec.ServiceAccount)
		if err = controllerutil.SetOwnerReference(as, serviceAccount, scheme); err != nil {
			return nil, errors.WithStack(err)
		}
		serviceAccountName = serviceAccount.Name
	}

	deployment, err := createArmadaServerDeployment(as, serviceAccountName, config)
	if err != nil {
		return nil, err
	}
	if err := controllerutil.SetOwnerReference(as, deployment, scheme); err != nil {
		return nil, err
	}

	ingressGrpc, err := createServerIngressGRPC(as, config)
	if err != nil {
		return nil, err
	}
	if ingressGrpc != nil {
		if err := controllerutil.SetOwnerReference(as, ingressGrpc, scheme); err != nil {
			return nil, err
		}
	}

	ingressHttp, err := createServerIngressHTTP(as, config)
	if err != nil {
		return nil, err
	}
	if ingressHttp != nil {
		if err := controllerutil.SetOwnerReference(as, ingressHttp, scheme); err != nil {
			return nil, err
		}
	}

	service := builders.Service(
		as.Name,
		as.Namespace,
		AllLabels(as.Name, as.Labels),
		IdentityLabel(as.Name),
		config,
		builders.ServiceEnableApplicationPortsOnly,
	)
	if err := controllerutil.SetOwnerReference(as, service, scheme); err != nil {
		return nil, err
	}
	profilingService, profilingIngress, err := newProfilingComponents(
		as,
		scheme,
		config,
		as.Spec.ProfilingIngressConfig,
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	pdb := createServerPodDisruptionBudget(as)
	if err := controllerutil.SetOwnerReference(as, pdb, scheme); err != nil {
		return nil, errors.WithStack(err)
	}

	var prometheusRule *monitoringv1.PrometheusRule
	var serviceMonitor *monitoringv1.ServiceMonitor
	if as.Spec.Prometheus != nil && as.Spec.Prometheus.Enabled {
		prometheusRule = createServerPrometheusRule(as)
		if err := controllerutil.SetOwnerReference(as, prometheusRule, scheme); err != nil {
			return nil, err
		}

		serviceMonitor = createServerServiceMonitor(as)
		if err := controllerutil.SetOwnerReference(as, serviceMonitor, scheme); err != nil {
			return nil, err
		}
	}

	jobs := []*batchv1.Job{{}}
	if as.Spec.PulsarInit {
		jobs, err = createArmadaServerMigrationJobs(as, config)
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
		IngressProfiling:    profilingIngress,
		Service:             service,
		ServiceProfiling:    profilingService,
		ServiceAccount:      serviceAccount,
		Secret:              secret,
		PodDisruptionBudget: pdb,
		PrometheusRule:      prometheusRule,
		ServiceMonitor:      serviceMonitor,
		Jobs:                jobs,
	}, nil

}

func createArmadaServerMigrationJobs(as *installv1alpha1.ArmadaServer, commonConfig *builders.CommonApplicationConfig) ([]*batchv1.Job, error) {
	terminationGracePeriodSeconds := as.Spec.TerminationGracePeriodSeconds
	parallelism := int32(1)
	completions := int32(1)
	backoffLimit := int32(0)

	appConfig, err := builders.ConvertRawExtensionToYaml(as.Spec.ApplicationConfig)
	if err != nil {
		return nil, err
	}
	var asConfig AppConfig
	err = yaml.Unmarshal([]byte(appConfig), &asConfig)
	if err != nil {
		return nil, err
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
					SecurityContext:               as.Spec.PodSecurityContext,
					Containers: []corev1.Container{{
						Name:            "wait-for-pulsar",
						ImagePullPolicy: corev1.PullIfNotPresent,
						Image:           defaultAlpineImage(),
						Args: []string{
							"/bin/sh",
							"-c",
							`echo "Waiting for Pulsar... ($PULSARHOST:$PULSARPORT)"
							while ! nc -z $PULSARHOST $PULSARPORT; do sleep 1; done
              echo "Pulsar started!"`,
						},
						Ports: []corev1.ContainerPort{{
							Name:          "metrics",
							ContainerPort: commonConfig.MetricsPort,
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
						SecurityContext: as.Spec.SecurityContext,
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
						ImagePullPolicy: corev1.PullIfNotPresent,
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
						SecurityContext: as.Spec.SecurityContext,
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

func createArmadaServerDeployment(
	as *installv1alpha1.ArmadaServer,
	serviceAccountName string,
	commonConfig *builders.CommonApplicationConfig,
) (*appsv1.Deployment, error) {
	env := createEnv(as.Spec.Environment)
	pulsarConfig, err := ExtractPulsarConfig(as.Spec.ApplicationConfig)
	if err != nil {
		return nil, err
	}
	volumes := createVolumes(as.Name, as.Spec.AdditionalVolumes)
	volumes = append(volumes, createPulsarVolumes(pulsarConfig)...)
	volumeMounts := createVolumeMounts(GetConfigFilename(as.Name), as.Spec.AdditionalVolumeMounts)
	volumeMounts = append(volumeMounts, createPulsarVolumeMounts(pulsarConfig)...)

	readinessProbe, livenessProbe := CreateProbesWithScheme(GetServerScheme(commonConfig.GRPC.TLS))

	deployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      as.Name,
			Namespace: as.Namespace,
			Labels:    AllLabels(as.Name, as.Labels),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: as.Spec.Replicas,
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
					ServiceAccountName:            serviceAccountName,
					TerminationGracePeriodSeconds: as.DeletionGracePeriodSeconds,
					SecurityContext:               as.Spec.PodSecurityContext,
					Affinity:                      defaultAffinity(as.Name, 100),
					Containers: []corev1.Container{{
						Name:            "armadaserver",
						ImagePullPolicy: corev1.PullIfNotPresent,
						Image:           ImageString(as.Spec.Image),
						Args:            []string{appConfigFlag, appConfigFilepath},
						Ports:           newContainerPortsAll(commonConfig),
						Env:             env,
						VolumeMounts:    volumeMounts,
						SecurityContext: as.Spec.SecurityContext,
						ReadinessProbe:  readinessProbe,
						LivenessProbe:   livenessProbe,
					}},
					Volumes: volumes,
				},
			},
			Strategy: defaultDeploymentStrategy(1),
		},
	}
	if as.Spec.Resources != nil {
		deployment.Spec.Template.Spec.Containers[0].Resources = *as.Spec.Resources
		deployment.Spec.Template.Spec.Containers[0].Env = addGoMemLimit(deployment.Spec.Template.Spec.Containers[0].Env, *as.Spec.Resources)
	}

	return &deployment, nil
}

func createServerIngressGRPC(as *installv1alpha1.ArmadaServer, config *builders.CommonApplicationConfig) (*networkingv1.Ingress, error) {
	if len(as.Spec.HostNames) == 0 {
		// if no hostnames, no ingress can be configured
		return nil, nil
	}

	name := as.Name + "-grpc"
	labels := AllLabels(as.Name, as.Spec.Labels, as.Spec.Ingress.Labels)
	baseAnnotations := map[string]string{
		"nginx.ingress.kubernetes.io/ssl-redirect": "true",
	}
	annotations := buildIngressAnnotations(as.Spec.Ingress, baseAnnotations, BackendProtocolGRPC, config.GRPC.Enabled)

	secretName := as.Name + "-service-tls"
	serviceName := as.Name
	servicePort := config.GRPCPort
	path := "/"
	ingress, err := builders.Ingress(name, as.Namespace, labels, annotations, as.Spec.HostNames, serviceName, secretName, path, servicePort)
	return ingress, errors.WithStack(err)
}

func createServerIngressHTTP(as *installv1alpha1.ArmadaServer, config *builders.CommonApplicationConfig) (*networkingv1.Ingress, error) {
	if len(as.Spec.HostNames) == 0 {
		// when no hostnames, no ingress can be configured
		return nil, nil
	}
	name := as.Name + "-rest"
	labels := AllLabels(as.Name, as.Spec.Labels, as.Spec.Ingress.Labels)
	baseAnnotations := map[string]string{
		"nginx.ingress.kubernetes.io/rewrite-target": "/$2",
		"nginx.ingress.kubernetes.io/ssl-redirect":   "true",
	}
	annotations := buildIngressAnnotations(as.Spec.Ingress, baseAnnotations, BackendProtocolHTTP, config.GRPC.Enabled)

	secretName := as.Name + "-service-tls"
	serviceName := as.Name
	servicePort := config.HTTPPort
	path := "/api(/|$)(.*)"
	ingress, err := builders.Ingress(name, as.Namespace, labels, annotations, as.Spec.HostNames, serviceName, secretName, path, servicePort)
	return ingress, errors.WithStack(err)
}

func createServerPodDisruptionBudget(as *installv1alpha1.ArmadaServer) *policyv1.PodDisruptionBudget {
	return &policyv1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{Name: as.Name, Namespace: as.Namespace},
		Spec:       policyv1.PodDisruptionBudgetSpec{},
		Status:     policyv1.PodDisruptionBudgetStatus{},
	}
}

func createServerServiceMonitor(as *installv1alpha1.ArmadaServer) *monitoringv1.ServiceMonitor {
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
func createServerPrometheusRule(server *installv1alpha1.ArmadaServer) *monitoringv1.PrometheusRule {
	queueSize := `max(sum(armada_queue_size) by (queueName, pod)) by (queueName) > 0`
	queuePriority := `avg(sum(armada_queue_priority) by (pool, queueName, pod)) by (pool, queueName)`
	queueIdeal := `(sum(armada:queue:resource:queued{resourceType="cpu"} > bool 0) by (queueName, pool) * (1 / armada:queue:priority))
               / ignoring(queueName) group_left sum(sum(armada:queue:resource:queued{resourceType="cpu"} > bool 0) by (queueName, pool) * (1 / armada:queue:priority)) by (pool) * 100`

	queueResourceQueued := `max(sum(armada_queue_resource_queued) by (pod, pool, queueName, resourceType)) by (pool, queueName, resourceType)`
	queueResourceAllocated := `max(sum(armada_queue_resource_allocated) by (pod, pool, cluster, queueName, resourceType, nodeType)) by (pool, cluster, queueName, resourceType, nodeType)`
	queueResourceUsed := `max(sum(armada_queue_resource_used) by (pod, pool, cluster, queueName, resourceType, nodeType)) by (pool, cluster, queueName, resourceType, nodeType)`
	serverHist := `histogram_quantile(0.95, sum(rate(grpc_server_handling_seconds_bucket{grpc_type!="server_stream"}[2m])) by (grpc_method,grpc_service, le))`
	serverRequestRate := `sum(rate(grpc_server_handled_total[2m])) by (grpc_method,grpc_service)`
	logRate := `sum(rate(log_messages[2m])) by (level)`
	availableCapacity := `avg(armada_cluster_available_capacity) by (pool, cluster, resourceType, nodeType)`
	resourceCapacity := `avg(armada_cluster_capacity) by (pool, cluster, resourceType, nodeType)`
	queuePodPhaseCount := `max(armada_queue_leased_pod_count) by (pool, cluster, queueName, phase, nodeType)`

	scrapeInterval := &metav1.Duration{Duration: defaultPrometheusInterval}
	if interval := server.Spec.Prometheus.ScrapeInterval; interval != nil {
		scrapeInterval = &metav1.Duration{Duration: interval.Duration}
	}
	durationString := duration.ShortHumanDuration(scrapeInterval.Duration)
	objectMetaName := "armada-" + server.Name + "-metrics"
	return &monitoringv1.PrometheusRule{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      server.Name,
			Namespace: server.Namespace,
			Labels:    AllLabels(server.Name, server.Labels, server.Spec.Prometheus.Labels),
		},
		Spec: monitoringv1.PrometheusRuleSpec{
			Groups: []monitoringv1.RuleGroup{{
				Name:     objectMetaName,
				Interval: ptr.To(monitoringv1.Duration(durationString)),
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
