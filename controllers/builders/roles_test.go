package builders

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateClusterRole(t *testing.T) {
	testcases := map[string]struct {
		name      string
		namespace string
	}{
		"Binoculars": {
			name:      "binoculars",
			namespace: "binoculars",
		},
	}

	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			got := CreateClusterRole(tc.name, tc.namespace)
			require.True(t, got.Name == tc.name)
			require.True(t, got.Name == tc.namespace)
		})
	}
}
