package install

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/pkg/errors"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	installv1alpha1 "github.com/armadaproject/armada-operator/apis/install/v1alpha1"
)

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

func AllLabels(name string, labels map[string]string) map[string]string {
	baseLabels := map[string]string{"release": name}
	additionalLabels := AdditionalLabels(labels)
	baseLabels = MergeMaps(baseLabels, additionalLabels)
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
