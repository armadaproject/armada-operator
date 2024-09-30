package builders

import (
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"
)

const (
	defaultHTTPPort      = 8080
	defaultGRPCPort      = 50051
	defaultMetricsPort   = 9000
	defaultProfilingPort = 1337
)

type CommonApplicationConfig struct {
	HTTPPort     int32           `json:"httpPort"`
	HTTPNodePort int32           `json:"httpNodePort,omitempty"`
	GRPCPort     int32           `json:"grpcPort"`
	GRPCNodePort int32           `json:"grpcNodePort,omitempty"`
	MetricsPort  int32           `json:"metricsPort"`
	Profiling    ProfilingConfig `json:"profiling"`
	GRPC         GRPCConfig      `json:"grpc"`
}

type GRPCConfig struct {
	Enabled bool      `json:"enabled"`
	TLS     TLSConfig `json:"tls"`
}

type TLSConfig struct {
	Enabled bool `json:"enabled"`
}

type ProfilingConfig struct {
	Port int32 `json:"port"`
}

// ParseCommonApplicationConfig parses the raw application config into a CommonApplicationConfig.
func ParseCommonApplicationConfig(rawAppConfig runtime.RawExtension) (*CommonApplicationConfig, error) {
	appConfig, err := ConvertRawExtensionToYaml(rawAppConfig)
	if err != nil {
		return nil, err
	}

	config := CommonApplicationConfig{
		HTTPPort:    defaultHTTPPort,
		GRPCPort:    defaultGRPCPort,
		MetricsPort: defaultMetricsPort,
		Profiling: ProfilingConfig{
			Port: defaultProfilingPort,
		},
	}
	if err = yaml.Unmarshal([]byte(appConfig), &config); err != nil {
		return nil, errors.WithStack(err)
	}

	return &config, nil
}

// ConvertRawExtensionToYaml converts a RawExtension input to Yaml
func ConvertRawExtensionToYaml(config runtime.RawExtension) (string, error) {
	yamlConfig, err := yaml.JSONToYAML(config.Raw)
	if err != nil {
		return "", err
	}

	return string(yamlConfig), nil
}
