package builders

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"k8s.io/apimachinery/pkg/runtime"
)

func TestBuildCommonApplicationConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    runtime.RawExtension
		expected *CommonApplicationConfig
		wantErr  bool
	}{
		{
			name:  "default empty application config",
			input: runtime.RawExtension{Raw: []byte(`{ }`)},
			expected: &CommonApplicationConfig{
				HTTPPort:    defaultHTTPPort,
				GRPCPort:    defaultGRPCPort,
				MetricsPort: defaultMetricsPort,
				Profiling: ProfilingConfig{
					Port: defaultProfilingPort,
				},
			},
		},
		{
			name:     "invalid application config",
			input:    runtime.RawExtension{Raw: []byte(`{"httpPort": 8081`)},
			expected: nil,
			wantErr:  true,
		},
		{
			name:  "partially override default application config",
			input: runtime.RawExtension{Raw: []byte(`{"httpPort": 1212, "profiling": { "port": 1111}}`)},
			expected: &CommonApplicationConfig{
				HTTPPort:    1212,
				GRPCPort:    50051,
				MetricsPort: 9000,
				Profiling: ProfilingConfig{
					Port: 1111,
				},
			},
		},
		{
			name: "valid config",
			input: runtime.RawExtension{
				Raw: []byte(`{"httpPort": 8081, "grpcPort": 50052, "metricsPort": 9001, "profiling": { "port": 1337} }`),
			},
			expected: &CommonApplicationConfig{
				HTTPPort:    8081,
				GRPCPort:    50052,
				MetricsPort: 9001,
				Profiling: ProfilingConfig{
					Port: 1337,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pc, err := ParseCommonApplicationConfig(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, tt.expected, pc)
		})
	}
}

func TestConvertRawExtensionToYaml(t *testing.T) {
	tests := []struct {
		name     string
		input    runtime.RawExtension
		expected string
		wantErr  bool
	}{
		{
			name:     "it converts runtime.RawExtension json to yaml",
			input:    runtime.RawExtension{Raw: []byte(`{ "test": { "foo": "bar" }}`)},
			expected: "test:\n  foo: bar\n",
		},
		{
			name:     "it converts complex runtime.RawExtension json to yaml",
			input:    runtime.RawExtension{Raw: []byte(`{ "test": {"foo": "bar"}, "test1": {"foo1": { "foo2": "bar2" }}}`)},
			expected: "test:\n  foo: bar\ntest1:\n  foo1:\n    foo2: bar2\n",
		},
		{
			name:     "it errors if runtime.RawExtension raw is malformed json",
			input:    runtime.RawExtension{Raw: []byte(`{ "foo": "bar" `)},
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := ConvertRawExtensionToYaml(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, tt.expected, output)
		})
	}
}
