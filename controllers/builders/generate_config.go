package builders

import (
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"
)

const (
	armadaConfigKey = "armada-config.yaml"
)

func GenerateArmadaConfig(config runtime.RawExtension) (map[string][]byte, error) {
	yaml, err := convertRawExtensionToYaml(config)
	if err != nil {
		return nil, err
	}
	return map[string][]byte{armadaConfigKey: []byte(yaml)}, nil
}

func convertRawExtensionToYaml(config runtime.RawExtension) (string, error) {
	yamlConfig, err := yaml.JSONToYAML(config.Raw)
	if err != nil {
		return "", err
	}

	return string(yamlConfig), nil
}
