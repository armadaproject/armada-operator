package builders

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestService(t *testing.T) {
	testcases := map[string]struct {
		name      string
		namespace string
		labels    map[string]string
	}{
		"Binoculars": {
			name:      "binoculars",
			namespace: "binoculars",
			labels:    map[string]string{"app": "binoculars"},
		},
	}

	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			got := Service(tc.name, tc.namespace, tc.labels)
			assert.True(t, got.Name == tc.name)
			assert.True(t, got.Namespace == tc.namespace)
		})
	}
}
