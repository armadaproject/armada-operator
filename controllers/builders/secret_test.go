package builders

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestGenerateSecret(t *testing.T) {
	testcases := map[string]struct {
		appConfig runtime.RawExtension
		name      string
		namespace string
	}{
		"Binoculars": {
			appConfig: runtime.RawExtension{},
			name:      "binoculars",
			namespace: "binoculars",
		},
	}

	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			got, err := CreateSecret(tc.appConfig, tc.name, tc.namespace)
			require.NoError(t, err)
			require.True(t, got.Name == tc.name)
			require.True(t, got.Name == tc.namespace)
		})
	}
}
