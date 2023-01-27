package install

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	installv1alpha1 "github.com/armadaproject/armada-operator/apis/install/v1alpha1"
	"github.com/pkg/errors"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

// appendEnv will append the CRD environment vars to the provided k8s EnvVar slice
func appendEnv(envVars []corev1.EnvVar, crdEnv []installv1alpha1.Environment) []corev1.EnvVar {
	for _, envVar := range crdEnv {
		envVars = append(envVars, corev1.EnvVar{Name: envVar.Name, Value: envVar.Value})
	}
	return envVars
}

// appendVolumes will append the CRD AdditionalVolumes to the provided k8s Volumes slice
func appendVolumes(volumes []corev1.Volume, crdVolumes []installv1alpha1.AdditionalVolume) []corev1.Volume {
	for _, crdVolume := range crdVolumes {
		volumes = append(volumes, corev1.Volume{Name: crdVolume.Name, VolumeSource: corev1.VolumeSource{Secret: &crdVolume.Secret}})
	}
	return volumes
}

// appendVolumeMountss will append the CRD AdditionalVolumeMounts  to the provided k8s VolumeMounts slice
func appendVolumeMounts(volumeMounts []corev1.VolumeMount, crdVolumeMounts []installv1alpha1.AdditionalVolumeMounts) []corev1.VolumeMount {
	for _, crdVolume := range crdVolumeMounts {
		volumeMounts = append(volumeMounts, crdVolume.Volume)
	}
	return volumeMounts
}
