package builders

import (
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"
)

// GenerateArmadaConfig generates armada config from the provided raw data and stores it into a map under the provided key.
func GenerateArmadaConfig(config runtime.RawExtension, key string) (map[string][]byte, error) {
	yml, err := convertRawExtensionToYaml(config)
	if err != nil {
		return nil, err
	}
	return map[string][]byte{key: []byte(yml)}, nil
}

func convertRawExtensionToYaml(config runtime.RawExtension) (string, error) {
	yamlConfig, err := yaml.JSONToYAML(config.Raw)
	if err != nil {
		return "", err
	}

	return string(yamlConfig), nil
}
