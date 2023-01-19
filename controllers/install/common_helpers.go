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
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Generate a docker image
func ImageString(image installv1alpha1.Image) string {
	return fmt.Sprintf("%s/%s", image.Repository, image.Tag)
}

// Utility for merging maps
// Annotations and Labels need to be merged from existing maps
func MergeMaps[M ~map[K]V, K comparable, V any](src ...M) M {
	merged := make(M)
	for _, m := range src {
		for k, v := range m {
			merged[k] = v
		}
	}
	return merged
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
func waitForJob(ctx context.Context, cl client.Client, job *batchv1.Job) (err error) {
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
		time.Sleep(time.Second * 5)
	}
}

// isJobFinished will assess if the job is finished (complete of failed).
func isJobFinished(job *batchv1.Job) bool {
	for _, condition := range job.Status.Conditions {
		if (condition.Type == batchv1.JobComplete || condition.Type == batchv1.JobFailed) && condition.Status == v1.ConditionTrue {
			return true
		}
	}

	return false
}
