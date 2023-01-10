package install

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	installv1alpha1 "github.com/armadaproject/armada-operator/apis/install/v1alpha1"
)

func getExecutorConfigFilename(executor *installv1alpha1.Executor) string {
	return getExecutorConfigName(executor) + ".yaml"
}

func getExecutorConfigName(executor *installv1alpha1.Executor) string {
	return fmt.Sprintf("%s-%s", executor.Name, "config")
}

func getExecutorChecksumConfig(executor *installv1alpha1.Executor) string {
	data := executor.Spec.ApplicationConfig.Raw
	sha := sha256.Sum256(data)
	return hex.EncodeToString(sha[:])
}

func getAllExecutorLabels(executor *installv1alpha1.Executor) map[string]string {
	baseLabels := map[string]string{"release": executor.Name}
	additionalLabels := getExecutorAdditionalLabels(executor)
	baseLabels = MergeMaps(baseLabels, additionalLabels)
	identityLabels := getExecutorIdentityLabels(executor)
	baseLabels = MergeMaps(baseLabels, identityLabels)
	return baseLabels
}

func getExecutorIdentityLabels(executor *installv1alpha1.Executor) map[string]string {
	return map[string]string{"app": executor.Name}
}

func getExecutorAdditionalLabels(executor *installv1alpha1.Executor) map[string]string {
	m := make(map[string]string, len(executor.Labels))
	for k, v := range executor.Labels {
		m[k] = v
	}
	return m
}

func MergeMaps[M ~map[K]V, K comparable, V any](src ...M) M {
	merged := make(M)
	for _, m := range src {
		for k, v := range m {
			merged[k] = v
		}
	}
	return merged
}
