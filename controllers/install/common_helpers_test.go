package install

import (
	"testing"

	install "github.com/armadaproject/armada-operator/apis/install/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestImageString(t *testing.T) {
	tests := []struct {
		name     string
		Image    install.Image
		expected string
	}{
		{
			name:     "Generate Image Name",
			Image:    install.Image{Repository: "blah", Tag: "tag"},
			expected: "blah/tag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := ImageString(tt.Image)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestAllLabels(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]string
		expected map[string]string
	}{
		{
			name:     "binoculars",
			input:    map[string]string{"hello": "world"},
			expected: map[string]string{"hello": "world", "app": "binoculars", "release": "binoculars"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := AllLabels(tt.name, tt.input)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestGetConfigName(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{
			name:     "binoculars",
			expected: "binoculars-config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := GetConfigName(tt.name)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestIdentityLabel(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "binoculars",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := IdentityLabel(tt.name)
			assert.Equal(t, actual["app"], tt.name)
		})
	}
}

func TestGenerateChecksumConfig(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name:     "binoculars",
			input:    []byte(`{ "test": { "foo": "bar" }}`),
			expected: "97503bec62eae4ddbc5da8c4e8743d580faf2178649fcc95be8a3f3af4ef09ca",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := GenerateChecksumConfig(tt.input)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
