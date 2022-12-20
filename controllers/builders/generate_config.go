package builders

import "gopkg.in/yaml.v2"

func generateArmadaConfig(config map[string]any) (map[string][]byte, error) {
	data, err := toYaml(config)
	if err != nil {
		return nil, err
	}
	return map[string][]byte{"armada-config.yaml": data}, nil
}

func toYaml(data map[string]any) ([]byte, error) {
	return yaml.Marshal(data)
}
