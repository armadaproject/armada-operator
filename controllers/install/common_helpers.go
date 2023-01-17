package install

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

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
