package install

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"reflect"
	"time"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/go-logr/logr"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	schedulingv1 "k8s.io/api/scheduling/v1"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	installv1alpha1 "github.com/armadaproject/armada-operator/api/install/v1alpha1"
	"github.com/armadaproject/armada-operator/internal/controller/builders"
)

const (
	// jobTimeout specifies the maximum time to wait for a job to complete.
	jobTimeout = time.Second * 120
	// jobPollInterval specifies the interval to poll for job completion.
	jobPollInterval = time.Second * 5
	// defaultPrometheusInterval is the default interval for Prometheus scraping.
	defaultPrometheusInterval = 1 * time.Second
	// appConfigFlag is the flag to specify the application config file in the container.
	appConfigFlag = "--config"
	// appConfigFilepath is the path to the application config file in the container.
	appConfigFilepath = "/config/application_config.yaml"
)

// CommonComponents are the base components for all Armada services
type CommonComponents struct {
	Deployment          *appsv1.Deployment
	IngressGrpc         *networkingv1.Ingress
	IngressHttp         *networkingv1.Ingress
	IngressProfiling    *networkingv1.Ingress
	Service             *corev1.Service
	ServiceProfiling    *corev1.Service
	ServiceAccount      *corev1.ServiceAccount
	Secret              *corev1.Secret
	ClusterRole         *rbacv1.ClusterRole
	ClusterRoleBindings []*rbacv1.ClusterRoleBinding
	PriorityClasses     []*schedulingv1.PriorityClass
	PrometheusRule      *monitoringv1.PrometheusRule
	ServiceMonitor      *monitoringv1.ServiceMonitor
	PodDisruptionBudget *policyv1.PodDisruptionBudget
	Jobs                []*batchv1.Job
	CronJob             *batchv1.CronJob
}

// CleanupFunc is a function that will clean up additional resources which are not deleted by owner references.
type CleanupFunc func(context.Context) error

// DeepCopy will deep-copy values from the receiver and return a new reference
func (cc *CommonComponents) DeepCopy() *CommonComponents {
	var clusterRoleBindings []*rbacv1.ClusterRoleBinding
	for _, crb := range cc.ClusterRoleBindings {
		clusterRoleBindings = append(clusterRoleBindings, crb.DeepCopy())
	}
	var priorityClasses []*schedulingv1.PriorityClass
	for _, pc := range cc.PriorityClasses {
		priorityClasses = append(priorityClasses, pc.DeepCopy())
	}
	var jobs []*batchv1.Job
	for _, job := range cc.Jobs {
		jobs = append(jobs, job.DeepCopy())
	}

	cloned := &CommonComponents{
		Deployment:          cc.Deployment.DeepCopy(),
		Service:             cc.Service.DeepCopy(),
		ServiceAccount:      cc.ServiceAccount.DeepCopy(),
		Secret:              cc.Secret.DeepCopy(),
		ClusterRole:         cc.ClusterRole.DeepCopy(),
		ClusterRoleBindings: clusterRoleBindings,
		PriorityClasses:     priorityClasses,
		Jobs:                jobs,
		CronJob:             cc.CronJob.DeepCopy(),
		ServiceMonitor:      cc.ServiceMonitor.DeepCopy(),
		PrometheusRule:      cc.PrometheusRule.DeepCopy(),
		IngressGrpc:         cc.IngressGrpc.DeepCopy(),
		IngressHttp:         cc.IngressHttp.DeepCopy(),
		PodDisruptionBudget: cc.PodDisruptionBudget.DeepCopy(),
	}

	return cloned
}

// ReconcileComponents will copy values from newComponents to the receiver
func (cc *CommonComponents) ReconcileComponents(newComponents *CommonComponents) {
	cc.Secret.Data = newComponents.Secret.Data
	cc.Secret.Labels = newComponents.Secret.Labels
	cc.Secret.Annotations = newComponents.Secret.Annotations
	cc.Deployment.Spec = newComponents.Deployment.Spec
	cc.Deployment.Labels = newComponents.Deployment.Labels
	cc.Deployment.Annotations = newComponents.Deployment.Annotations

	if newComponents.Service != nil {
		cc.Service.Spec = newComponents.Service.Spec
		cc.Service.Labels = newComponents.Service.Labels
		cc.Service.Annotations = newComponents.Service.Annotations
	} else {
		cc.Service = nil
	}

	if newComponents.ServiceProfiling != nil {
		cc.ServiceProfiling.Spec = newComponents.ServiceProfiling.Spec
		cc.ServiceProfiling.Labels = newComponents.ServiceProfiling.Labels
		cc.ServiceProfiling.Annotations = newComponents.ServiceProfiling.Annotations
	} else {
		cc.ServiceProfiling = nil
	}

	if newComponents.ServiceAccount != nil {
		cc.ServiceAccount.Labels = newComponents.ServiceAccount.Labels
		cc.ServiceAccount.Annotations = newComponents.ServiceAccount.Annotations
		cc.ServiceAccount.ImagePullSecrets = newComponents.ServiceAccount.ImagePullSecrets
		cc.ServiceAccount.Secrets = newComponents.ServiceAccount.Secrets
		cc.ServiceAccount.AutomountServiceAccountToken = newComponents.ServiceAccount.AutomountServiceAccountToken
	} else {
		cc.ServiceAccount = nil
	}

	if newComponents.ClusterRole != nil {
		cc.ClusterRole.Rules = newComponents.ClusterRole.Rules
		cc.ClusterRole.Labels = newComponents.ClusterRole.Labels
		cc.ClusterRole.Annotations = newComponents.ClusterRole.Annotations
	} else {
		cc.ClusterRole = nil
	}

	if newComponents.IngressGrpc != nil {
		cc.IngressGrpc.Spec = newComponents.IngressGrpc.Spec
		cc.IngressGrpc.Labels = newComponents.IngressGrpc.Labels
		cc.IngressGrpc.Annotations = newComponents.IngressGrpc.Annotations
	} else {
		cc.IngressGrpc = nil
	}

	if newComponents.IngressHttp != nil {
		cc.IngressHttp.Spec = newComponents.IngressHttp.Spec
		cc.IngressHttp.Labels = newComponents.IngressHttp.Labels
		cc.IngressHttp.Annotations = newComponents.IngressHttp.Annotations
	} else {
		cc.IngressHttp = nil
	}

	if newComponents.IngressProfiling != nil {
		cc.IngressProfiling.Spec = newComponents.IngressProfiling.Spec
		cc.IngressProfiling.Labels = newComponents.IngressProfiling.Labels
		cc.IngressProfiling.Annotations = newComponents.IngressProfiling.Annotations
	} else {
		cc.IngressProfiling = nil
	}

	if newComponents.PodDisruptionBudget != nil {
		cc.PodDisruptionBudget.Spec = newComponents.PodDisruptionBudget.Spec
		cc.PodDisruptionBudget.Labels = newComponents.PodDisruptionBudget.Labels
		cc.PodDisruptionBudget.Annotations = newComponents.PodDisruptionBudget.Annotations
	} else {
		cc.PodDisruptionBudget = nil
	}

	for i := range cc.ClusterRoleBindings {
		cc.ClusterRoleBindings[i].RoleRef = newComponents.ClusterRoleBindings[i].RoleRef
		cc.ClusterRoleBindings[i].Subjects = newComponents.ClusterRoleBindings[i].Subjects
		cc.ClusterRoleBindings[i].Labels = newComponents.ClusterRoleBindings[i].Labels
		cc.ClusterRoleBindings[i].Annotations = newComponents.ClusterRoleBindings[i].Annotations
	}
	for i := range cc.PriorityClasses {
		cc.PriorityClasses[i].PreemptionPolicy = newComponents.PriorityClasses[i].PreemptionPolicy
		cc.PriorityClasses[i].Value = newComponents.PriorityClasses[i].Value
		cc.PriorityClasses[i].Description = newComponents.PriorityClasses[i].Description
		cc.PriorityClasses[i].GlobalDefault = newComponents.PriorityClasses[i].GlobalDefault
		cc.PriorityClasses[i].Labels = newComponents.PriorityClasses[i].Labels
		cc.PriorityClasses[i].Annotations = newComponents.PriorityClasses[i].Annotations
	}

	if newComponents.CronJob != nil {
		cc.CronJob.Spec = newComponents.CronJob.Spec
		cc.CronJob.Annotations = newComponents.CronJob.Annotations
		cc.CronJob.Labels = newComponents.CronJob.Labels
	} else {
		cc.CronJob = nil
	}
}

// PostgresConfig is used for scanning postgres section of application config
type PostgresConfig struct {
	Connection ConnectionConfig
}

// ConnectionConfig is used for scanning connection section of postgres config
type ConnectionConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Dbname   string
}

// PulsarConfig is used for scanning pulsar section of application config
type PulsarConfig struct {
	ArmadaInit            ArmadaInit
	AuthenticationEnabled bool
	TlsEnabled            bool
	AuthenticationSecret  string
	Cacert                string
}

// ArmadaInit used to initialize pulsar
type ArmadaInit struct {
	Enabled    bool
	Image      Image
	BrokerHost string
	Protocol   string
	AdminPort  int
	Port       int
}

// Image represents a docker image
type Image struct {
	Repository string
	Tag        string
}

// AppConfig is used for scanning the appconfig to find particular values
type AppConfig struct {
	Pulsar PulsarConfig
}

// ImageString generates a docker image.
func ImageString(image installv1alpha1.Image) string {
	return fmt.Sprintf("%s:%s", image.Repository, image.Tag)
}

// MergeMaps is a utility for merging maps.
// Annotations and Labels need to be merged from existing maps.
func MergeMaps[M ~map[K]V, K comparable, V any](src ...M) M {
	merged := make(M)
	for _, m := range src {
		for k, v := range m {
			merged[k] = v
		}
	}
	return merged
}

func GetConfigFilename(name string) string {
	return fmt.Sprintf("%s.yaml", GetConfigName(name))
}

func GetConfigName(name string) string {
	return fmt.Sprintf("%s-config", name)
}

func GenerateChecksumConfig(data []byte) string {
	sha := sha256.Sum256(data)
	return hex.EncodeToString(sha[:])
}

func IdentityLabel(name string) map[string]string {
	return map[string]string{"app": name}
}

func AdditionalLabels(label map[string]string) map[string]string {
	m := make(map[string]string, len(label))
	for k, v := range label {
		m[k] = v
	}
	return m
}

func AllLabels(name string, labelMaps ...map[string]string) map[string]string {
	baseLabels := map[string]string{"release": name}
	for _, labels := range labelMaps {
		if labels == nil {
			continue
		}
		additionalLabels := AdditionalLabels(labels)
		baseLabels = MergeMaps(baseLabels, additionalLabels)
	}
	identityLabels := IdentityLabel(name)
	baseLabels = MergeMaps(baseLabels, identityLabels)
	return baseLabels
}

// ExtractPulsarConfig will unmarshal the appconfig and return the PulsarConfig portion
func ExtractPulsarConfig(config runtime.RawExtension) (PulsarConfig, error) {
	appConfig, err := builders.ConvertRawExtensionToYaml(config)
	if err != nil {
		return PulsarConfig{}, err
	}
	var asConfig AppConfig
	err = yaml.Unmarshal([]byte(appConfig), &asConfig)
	if err != nil {
		return PulsarConfig{}, err
	}
	return asConfig.Pulsar, nil
}

// waitForJob waits for the Job to reach a terminal state (complete or failed).
func waitForJob(ctx context.Context, c client.Client, job *batchv1.Job, pollInterval, timeout time.Duration) error {
	return wait.PollUntilContextTimeout(
		ctx,
		pollInterval,
		timeout,
		false,
		func(ctx context.Context) (bool, error) {
			key := client.ObjectKeyFromObject(job)
			if err := c.Get(ctx, key, job); err != nil {
				return false, err
			}
			return isJobFinished(job), nil
		})
}

// isJobFinished will assess if the job is finished (complete of failed).
func isJobFinished(job *batchv1.Job) bool {
	for _, condition := range job.Status.Conditions {
		if (condition.Type == batchv1.JobComplete || condition.Type == batchv1.JobFailed) && condition.Status == corev1.ConditionTrue {
			return true
		}
	}

	return false
}

// createEnv creates the default EnvVars and appends the CRD environment vars
func createEnv(crdEnv []corev1.EnvVar) []corev1.EnvVar {
	envVars := []corev1.EnvVar{
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
	envVars = append(envVars, crdEnv...)
	return envVars
}

// createVolumes creates the default appconfig Volume and appends the CRD AdditionalVolumes
func createVolumes(configVolumeSecretName string, crdVolumes []corev1.Volume) []corev1.Volume {
	volumes := []corev1.Volume{{
		Name: volumeConfigKey,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: configVolumeSecretName,
			},
		},
	}}
	volumes = append(volumes, crdVolumes...)
	return volumes
}

// createVolumeMounts creates the app config VolumeMount and appends the CRD AdditionalVolumeMounts
func createVolumeMounts(configVolumeSecretName string, crdVolumeMounts []corev1.VolumeMount) []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      volumeConfigKey,
			ReadOnly:  true,
			MountPath: "/config/application_config.yaml",
			SubPath:   configVolumeSecretName,
		},
	}
	volumeMounts = append(volumeMounts, crdVolumeMounts...)
	return volumeMounts
}

// createPulsarVolumeMounts creates the pulsar volumeMounts for token and/or cert
func createPulsarVolumeMounts(pulsarConfig PulsarConfig) []corev1.VolumeMount {
	var volumeMounts []corev1.VolumeMount
	if pulsarConfig.AuthenticationEnabled {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "pulsar-token",
			ReadOnly:  true,
			MountPath: "/pulsar/tokens",
		})
	}
	if pulsarConfig.TlsEnabled {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "pulsar-ca",
			ReadOnly:  true,
			MountPath: "/pulsar/ca",
		})
	}
	return volumeMounts
}

// createPulsarVolumes creates the pulsar volumes for token and/or cert
func createPulsarVolumes(pulsarConfig PulsarConfig) []corev1.Volume {
	var volumes []corev1.Volume
	if pulsarConfig.AuthenticationEnabled {
		secretName := "armada-pulsar-token-armada-admin"
		if pulsarConfig.AuthenticationSecret != "" {
			secretName = pulsarConfig.AuthenticationSecret
		}
		volumes = append(volumes, corev1.Volume{
			Name: "pulsar-token",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: secretName,
					Items: []corev1.KeyToPath{{
						Key:  "TOKEN",
						Path: "pulsar-token",
					}},
				},
			},
		})
	}
	if pulsarConfig.TlsEnabled {
		secretName := "armada-pulsar-ca-tls"
		if pulsarConfig.Cacert != "" {
			secretName = pulsarConfig.Cacert
		}
		volumes = append(volumes, corev1.Volume{
			Name: "pulsar-ca",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: secretName,
					Items: []corev1.KeyToPath{{
						Key:  "ca.crt",
						Path: "ca.crt",
					}},
				},
			},
		})
	}
	return volumes
}

// addGoMemLimit will add the GOMEMLIMIT environment variable if the memory limit is set.
func addGoMemLimit(env []corev1.EnvVar, resources corev1.ResourceRequirements) []corev1.EnvVar {
	if resources.Limits.Memory() != nil && resources.Limits.Memory().Value() != 0 {
		val := resources.Limits.Memory().Value()
		goMemLimit := corev1.EnvVar{Name: "GOMEMLIMIT", Value: fmt.Sprintf("%dB", val)}
		env = append(env, goMemLimit)
	}
	return env
}

type BackendProtocol string

const (
	BackendProtocolGRPC BackendProtocol = "GRPC"
	BackendProtocolHTTP BackendProtocol = "HTTP"
)

func buildIngressAnnotations(
	ingressConfig *installv1alpha1.IngressConfig,
	baseAnnotations map[string]string,
	protocol BackendProtocol,
	useTLS bool,
) map[string]string {
	annotations := map[string]string{
		"kubernetes.io/ingress.class": ingressConfig.IngressClass,
	}
	if useTLS {
		annotations["nginx.ingress.kubernetes.io/backend-protocol"] = string(protocol) + "S"
		annotations["nginx.ingress.kubernetes.io/ssl-passthrough"] = "true"
	} else {
		annotations["nginx.ingress.kubernetes.io/backend-protocol"] = string(protocol)
	}
	for key, value := range baseAnnotations {
		annotations[key] = value
	}
	if ingressConfig.ClusterIssuer != "" {
		annotations["certmanager.k8s.io/cluster-issuer"] = ingressConfig.ClusterIssuer
		annotations["cert-manager.io/cluster-issuer"] = ingressConfig.ClusterIssuer
	}
	for key, value := range ingressConfig.Annotations {
		annotations[key] = value
	}
	return annotations
}

// checkAndHandleObjectDeletion handles the deletion of the resource by adding/removing the finalizer.
// If the resource is being deleted, it will remove the finalizer.
// If the resource is not being deleted, it will add the finalizer.
// If finish is true, the reconciliation should finish early.
func checkAndHandleObjectDeletion(
	ctx context.Context,
	r client.Client,
	object client.Object,
	finalizer string,
	cleanupF CleanupFunc,
	logger logr.Logger,
) (finish bool, err error) {
	logger = logger.WithValues("finalizer", finalizer)
	deletionTimestamp := object.GetDeletionTimestamp()
	if deletionTimestamp.IsZero() {
		// The object is not being deleted as deletionTimestamp.
		// In this case, we should add the finalizer if it is not already present.
		if err := addFinalizerIfNeeded(ctx, r, object, finalizer, logger); err != nil {
			return true, err
		}
	} else {
		// The object is being deleted so we should run the cleanup function if needed and remove the finalizer.
		return handleObjectDeletion(ctx, r, object, finalizer, cleanupF, logger)
	}
	// The object is not being deleted, continue reconciliation
	return false, nil
}

// addFinalizerIfNeeded will add the finalizer to the object if it is not already present.
func addFinalizerIfNeeded(
	ctx context.Context,
	client client.Client,
	object client.Object,
	finalizer string,
	logger logr.Logger,
) error {
	if !controllerutil.ContainsFinalizer(object, finalizer) {
		logger.Info("Attaching cleanup finalizer because object does not have a deletion timestamp set")
		controllerutil.AddFinalizer(object, finalizer)
		return client.Update(ctx, object)
	}
	return nil
}

func handleObjectDeletion(
	ctx context.Context,
	client client.Client,
	object client.Object,
	finalizer string,
	cleanupF CleanupFunc,
	logger logr.Logger,
) (finish bool, err error) {
	deletionTimestamp := object.GetDeletionTimestamp()
	logger.Info(
		"Object is being deleted as it has a non-zero deletion timestamp set",
		"deletionTimestamp", deletionTimestamp,
	)
	logger.Info(
		"Namespace-scoped objects will be deleted by Kubernetes based on their OwnerReference",
		"deletionTimestamp", deletionTimestamp,
	)
	// The object is being deleted
	if controllerutil.ContainsFinalizer(object, finalizer) {
		// Run additional cleanup function if it is provided
		if cleanupF != nil {
			if err := cleanupF(ctx); err != nil {
				return true, err
			}
		}
		// Remove our finalizer from the list and update it.
		logger.Info("Removing cleanup finalizer from object")
		controllerutil.RemoveFinalizer(object, finalizer)
		if err := client.Update(ctx, object); err != nil {
			return true, err
		}
	}

	// Stop reconciliation as the item is being deleted
	return true, nil
}

// upsertObjectIfNeeded will create or update the object with the mutateFn if the resource is not nil.
func upsertObjectIfNeeded(
	ctx context.Context,
	client client.Client,
	object client.Object,
	componentName string,
	mutateFn controllerutil.MutateFn,
	logger logr.Logger,
) error {
	if isNil(object) {
		return nil
	}

	logger.Info(fmt.Sprintf("Upserting %s %s object", componentName, object.GetObjectKind()))
	_, err := controllerutil.CreateOrUpdate(ctx, client, object, mutateFn)
	return err
}

// Helper function to determine if the object is nil even if it's a pointer to a nil value
func isNil(i any) bool {
	iv := reflect.ValueOf(i)
	if !iv.IsValid() {
		return true
	}
	switch iv.Kind() {
	case reflect.Ptr, reflect.Slice, reflect.Map, reflect.Func, reflect.Interface:
		return iv.IsNil()
	default:
		return false
	}
}

// deleteObjectIfNeeded will delete the object if it exists.
func deleteObjectIfNeeded(
	ctx context.Context,
	client client.Client,
	object client.Object,
	componentName string,
	logger logr.Logger,
) error {
	if isNil(object) {
		return nil
	}

	err := client.Delete(ctx, object)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil // nothing to do
		} else {
			return err
		}
	}

	logger.Info("Successfully deleted %s %s", componentName, object.GetObjectKind())

	return nil
}

// getObject will get the object from Kubernetes and return if it is missing or an error.
func getObject(
	ctx context.Context,
	client client.Client,
	object client.Object,
	namespacedName types.NamespacedName,
	logger logr.Logger,
) (miss bool, err error) {
	logger.Info("Fetching object from cache")
	if err := client.Get(ctx, namespacedName, object); err != nil {
		if k8serrors.IsNotFound(err) {
			logger.Info("Object not found in cache, ending reconcile...")
			return true, nil
		}
		return true, err
	}
	return false, nil
}

func newProfilingComponents(
	object metav1.Object,
	scheme *runtime.Scheme,
	commonConfig *builders.CommonApplicationConfig,
	ingressConfig *installv1alpha1.IngressConfig,
) (*corev1.Service, *networkingv1.Ingress, error) {
	profilingService, err := newProfilingService(object, commonConfig, scheme)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "error creating profiling service")
	}
	profilingIngress, err := newProfilingIngress(object, commonConfig, ingressConfig, scheme)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "error creating profiling ingress")
	}

	return profilingService, profilingIngress, nil
}

// newProfilingService creates a new Kubernetes Service for the profiling server and sets the owner reference to the parent object.
func newProfilingService(
	object metav1.Object,
	commonConfig *builders.CommonApplicationConfig,
	scheme *runtime.Scheme,
) (*corev1.Service, error) {
	profilingService := builders.Service(
		object.GetName()+"-profiling",
		object.GetNamespace(),
		AllLabels(object.GetName(), object.GetLabels()),
		IdentityLabel(object.GetName()),
		commonConfig,
		builders.ServiceEnableProfilingPortOnly,
	)
	if err := controllerutil.SetOwnerReference(object, profilingService, scheme); err != nil {
		return nil, err
	}

	return profilingService, nil
}

// newProfilingIngress creates a new Kubernetes Ingress for the profiling server and sets the owner reference to the parent object.
func newProfilingIngress(
	object metav1.Object,
	commonConfig *builders.CommonApplicationConfig,
	ingressConfig *installv1alpha1.IngressConfig,
	scheme *runtime.Scheme,
) (*networkingv1.Ingress, error) {
	if ingressConfig == nil {
		return nil, nil
	}
	baseAnnotations := map[string]string{
		"nginx.ingress.kubernetes.io/ssl-redirect": "true",
	}
	annotations := buildIngressAnnotations(ingressConfig, baseAnnotations, BackendProtocolHTTP, false)
	secretName := object.GetName() + "-service-tls"
	serviceName := object.GetName()
	servicePort := commonConfig.HTTPPort
	path := "/"
	profilingIngress, err := builders.Ingress(
		object.GetName()+"-profiling",
		object.GetNamespace(),
		AllLabels(object.GetName(), object.GetLabels(), ingressConfig.Labels),
		annotations,
		ingressConfig.Hostnames,
		serviceName,
		secretName,
		path,
		servicePort,
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if err := controllerutil.SetOwnerReference(object, profilingIngress, scheme); err != nil {
		return nil, err
	}

	return profilingIngress, nil
}

// newContainerPortsAll creates container ports for grpc, http and metrics server and optional port for profiling server.
func newContainerPortsAll(config *builders.CommonApplicationConfig) []corev1.ContainerPort {
	ports := []corev1.ContainerPort{newContainerPortGRPC(config), newContainerPortHTTP(config), newContainerPortMetrics(config)}
	if config.Profiling.Port > 0 {
		ports = append(ports, newContainerPortProfiling(config))
	}
	return ports
}

func newContainerPortsHTTPWithMetrics(config *builders.CommonApplicationConfig) []corev1.ContainerPort {
	ports := []corev1.ContainerPort{newContainerPortHTTP(config), newContainerPortMetrics(config)}
	if config.Profiling.Port > 0 {
		ports = append(ports, newContainerPortProfiling(config))
	}
	return ports
}

// newContainerPortsGRPCWithMetrics creates container ports for grpc and metrics server and optional port for profiling server.
func newContainerPortsGRPCWithMetrics(config *builders.CommonApplicationConfig) []corev1.ContainerPort {
	ports := []corev1.ContainerPort{newContainerPortGRPC(config), newContainerPortMetrics(config)}
	if config.Profiling.Port > 0 {
		ports = append(ports, newContainerPortProfiling(config))
	}
	return ports
}

// newContainerPortsMetrics creates container ports for metrics server and optional port for profiling server.
func newContainerPortsMetrics(config *builders.CommonApplicationConfig) []corev1.ContainerPort {
	ports := []corev1.ContainerPort{newContainerPortMetrics(config)}
	if config.Profiling.Port > 0 {
		ports = append(ports, newContainerPortProfiling(config))
	}
	return ports
}

// newContainerPortGRPC creates a container port for grpc server from settings defined in builders.CommonApplicationConfig.
func newContainerPortGRPC(config *builders.CommonApplicationConfig) corev1.ContainerPort {
	return corev1.ContainerPort{
		Name:          "grpc",
		ContainerPort: config.GRPCPort,
		Protocol:      corev1.ProtocolTCP,
	}
}

// newContainerPortHTTP creates a container port for http server from settings defined in builders.CommonApplicationConfig.
func newContainerPortHTTP(config *builders.CommonApplicationConfig) corev1.ContainerPort {
	return corev1.ContainerPort{

		Name:          "http",
		ContainerPort: config.HTTPPort,
		Protocol:      corev1.ProtocolTCP,
	}
}

// newContainerPortMetrics creates a container port for metrics server from settings defined in builders.CommonApplicationConfig.
func newContainerPortMetrics(config *builders.CommonApplicationConfig) corev1.ContainerPort {
	return corev1.ContainerPort{
		Name:          "metrics",
		ContainerPort: config.MetricsPort,
		Protocol:      corev1.ProtocolTCP,
	}
}

func newContainerPortProfiling(config *builders.CommonApplicationConfig) corev1.ContainerPort {
	return corev1.ContainerPort{
		Name:          "profiling",
		ContainerPort: config.Profiling.Port,
		Protocol:      corev1.ProtocolTCP,
	}
}

func defaultAffinity(app string, weight int32) *corev1.Affinity {
	return &corev1.Affinity{
		PodAffinity: &corev1.PodAffinity{
			PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{{
				Weight: weight,
				PodAffinityTerm: corev1.PodAffinityTerm{
					TopologyKey: "kubernetes.io/hostname",
					LabelSelector: &metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{{
							Key:      "app",
							Operator: metav1.LabelSelectorOpIn,
							Values:   []string{app},
						}},
					},
				},
			}},
		},
	}
}

func defaultDeploymentStrategy(maxUnavailable int32) appsv1.DeploymentStrategy {
	return appsv1.DeploymentStrategy{
		Type: appsv1.RollingUpdateDeploymentStrategyType,
		RollingUpdate: &appsv1.RollingUpdateDeployment{
			MaxUnavailable: &intstr.IntOrString{IntVal: maxUnavailable},
		},
	}
}
