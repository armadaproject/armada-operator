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
