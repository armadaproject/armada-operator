package builders

import (
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"
)

const (
	armadaConfigKey = "armada-config.yaml"
)

func GenerateArmadaConfig(config runtime.RawExtension) (map[string][]byte, error) {
	yamlConfig, err := yaml.JSONToYAML(config.Raw)
	if err != nil {
		return nil, err
	}

	return map[string][]byte{armadaConfigKey: yamlConfig}, nil
}
