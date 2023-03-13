package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"k8s.io/apimachinery/pkg/runtime"
)

func TestBuildPortConfig(t *testing.T) {
	tests := []struct {
		name     string
		input    runtime.RawExtension
		expected PortConfig
		wantErr  bool
	}{
		{
			name:  "it provides some reasonable defaults",
			input: runtime.RawExtension{Raw: []byte(`{ }`)},
			expected: PortConfig{
				HttpPort:    8080,
				GrpcPort:    50051,
				MetricsPort: 9000,
			},
		},
		{
			name:  "it errors with bad json (so does everything else in the app)",
			input: runtime.RawExtension{Raw: []byte(`{"httpPort": 8081`)},
			expected: PortConfig{},
			wantErr: true,
		},
		{
			name:  "it accepts partial overrides from the config",
			input: runtime.RawExtension{Raw: []byte(`{"httpPort": 8081}`)},
			expected: PortConfig{
				HttpPort:    8081,
				GrpcPort:    50051,
				MetricsPort: 9000,
			},
		},
		{
			name: "it accepts complete override from the config",
			input: runtime.RawExtension{
				Raw: []byte(`{"httpPort": 8081, "grpcPort": 50052, "metricsPort": 9001 }`),
			},
			expected: PortConfig{
				HttpPort:    8081,
				GrpcPort:    50052,
				MetricsPort: 9001,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := CommonSpecBase{
				ApplicationConfig: tt.input,
			}
			err := c.BuildPortConfig()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, tt.expected, c.PortConfig)
		})
	}
}

func Test_convertRawExtensionToYaml(t *testing.T) {

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
			output, err := convertRawExtensionToYaml(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, tt.expected, output)
		})
	}
}
