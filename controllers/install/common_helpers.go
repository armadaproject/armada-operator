package install

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/pkg/errors"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	schedulingv1 "k8s.io/api/scheduling/v1"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	installv1alpha1 "github.com/armadaproject/armada-operator/apis/install/v1alpha1"
)

const (
	defaultPrometheusInterval = 1 * time.Second
)

// CommonComponents are the base components for all of the Armada services
type CommonComponents struct {
	Deployment          *appsv1.Deployment
	IngressGrpc         *networkingv1.Ingress
	IngressHttp         *networkingv1.Ingress
	Service             *corev1.Service
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

// DeepCopy will deep copy values from the receiver and return a new reference
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
func (oldComponents *CommonComponents) ReconcileComponents(newComponents *CommonComponents) {
	oldComponents.Secret.Data = newComponents.Secret.Data
	oldComponents.Secret.Labels = newComponents.Secret.Labels
	oldComponents.Secret.Annotations = newComponents.Secret.Annotations
	oldComponents.Deployment.Spec = newComponents.Deployment.Spec
	oldComponents.Deployment.Labels = newComponents.Deployment.Labels
	oldComponents.Deployment.Annotations = newComponents.Deployment.Annotations
	if newComponents.Service != nil {
		oldComponents.Service.Spec = newComponents.Service.Spec
		oldComponents.Service.Labels = newComponents.Service.Labels
		oldComponents.Service.Annotations = newComponents.Service.Annotations
	} else {
		oldComponents.Service = nil
	}

	if newComponents.ClusterRole != nil {
		oldComponents.ClusterRole.Rules = newComponents.ClusterRole.Rules
		oldComponents.ClusterRole.Labels = newComponents.ClusterRole.Labels
		oldComponents.ClusterRole.Annotations = newComponents.ClusterRole.Annotations
	} else {
		oldComponents.ClusterRole = nil
	}

	if newComponents.IngressGrpc != nil {
		oldComponents.IngressGrpc.Spec = newComponents.IngressGrpc.Spec
		oldComponents.IngressGrpc.Labels = newComponents.IngressGrpc.Labels
		oldComponents.IngressGrpc.Annotations = newComponents.IngressGrpc.Annotations
	} else {
		oldComponents.IngressGrpc = nil
	}

	if newComponents.IngressHttp != nil {
		oldComponents.IngressHttp.Spec = newComponents.IngressHttp.Spec
		oldComponents.IngressHttp.Labels = newComponents.IngressHttp.Labels
		oldComponents.IngressHttp.Annotations = newComponents.IngressHttp.Annotations
	} else {
		oldComponents.IngressHttp = nil
	}

	if newComponents.PodDisruptionBudget != nil {
		oldComponents.PodDisruptionBudget.Spec = newComponents.PodDisruptionBudget.Spec
		oldComponents.PodDisruptionBudget.Labels = newComponents.PodDisruptionBudget.Labels
		oldComponents.PodDisruptionBudget.Annotations = newComponents.PodDisruptionBudget.Annotations
	} else {
		oldComponents.PodDisruptionBudget = nil
	}

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

func AllLabels(name string, labels ...map[string]string) map[string]string {
	baseLabels := map[string]string{"release": name}
	for _, labelSet := range labels {
		additionalLabels := AdditionalLabels(labelSet)
		baseLabels = MergeMaps(baseLabels, additionalLabels)
	}
	identityLabels := IdentityLabel(name)
	baseLabels = MergeMaps(baseLabels, identityLabels)
	return baseLabels
}

// waitForJob will wait for some resolution of the job. Provide context with timeout if needed.
func waitForJob(ctx context.Context, cl client.Client, job *batchv1.Job, sleepTime time.Duration) (err error) {
	for {
		select {
		case <-ctx.Done():
			return errors.New("context timeout while waiting for job")
		default:
			key := client.ObjectKeyFromObject(job)
			err = cl.Get(ctx, key, job)
			if err != nil {
				return err
			}
			if isJobFinished(job) {
				return nil
			}
		}
		time.Sleep(sleepTime)
	}
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

// createVolumeMounts creates the appconfig VolumeMount and appends the CRD AdditionalVolumeMounts
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

// createPrometheusRule will provide a prometheus monitoring rule for the name and scrapeInterval
func createPrometheusRule(name, namespace string, scrapeInterval *metav1.Duration, labels ...map[string]string) *monitoringv1.PrometheusRule {
	if scrapeInterval == nil {
		scrapeInterval = &metav1.Duration{Duration: defaultPrometheusInterval}
	}
	restRequestHistogram := `histogram_quantile(0.95, ` +
		`sum(rate(rest_client_request_duration_seconds_bucket{service="` + name + `"}[2m])) by (endpoint, verb, url, le))`
	logRate := "sum(rate(log_messages[2m])) by (level)"
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
						Record: "armada:" + name + ":rest:request:histogram95",
						Expr:   intstr.IntOrString{StrVal: restRequestHistogram},
					},
					{
						Record: "armada:" + name + ":log:rate",
						Expr:   intstr.IntOrString{StrVal: logRate},
					},
				},
			}},
		},
	}
}
