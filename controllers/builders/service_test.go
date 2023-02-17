package builders

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestService(t *testing.T) {
	testcases := map[string]struct {
		name          string
		namespace     string
		labels        map[string]string
		identityLabel map[string]string
		ports         []corev1.ServicePort
	}{
		"Binoculars": {
			name:          "binoculars",
			namespace:     "binoculars",
			labels:        map[string]string{"app": "binoculars", "hello": "world"},
			identityLabel: map[string]string{"app": "binoculars"},
			ports: []corev1.ServicePort{
				{
					Name: "grpc",
					Port: 50059,
				},
				{
					Name: "web",
					Port: 8080,
				},
				{
					Name: "metrics",
					Port: 9000,
				},
			}},
		"Lookout": {
			name:          "lookout",
			namespace:     "lookout",
			labels:        map[string]string{"app": "lookout", "hello": "world"},
			identityLabel: map[string]string{"app": "binoculars"},
			ports: []corev1.ServicePort{
				{
					Name: "grpc",
					Port: 50059,
				},
				{
					Name: "web",
					Port: 8080,
				},
				{
					Name: "metrics",
					Port: 9000,
				},
			},
		}}

	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			got := Service(tc.name, tc.namespace, tc.labels, tc.identityLabel, tc.ports)
			assert.True(t, got.Name == tc.name)
			assert.True(t, got.Namespace == tc.namespace)
		})
	}
}
