package builders

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"

	"github.com/armadaproject/armada-operator/apis/install/v1alpha1"
)

func TestService(t *testing.T) {
	testcases := map[string]struct {
		name          string
		namespace     string
		labels        map[string]string
		identityLabel map[string]string
		portConfig    v1alpha1.PortConfig
		ports         []corev1.ServicePort
	}{
		"PortConfig values are translated to ServicePort": {
			name:          "lookout",
			namespace:     "lookout",
			labels:        map[string]string{"app": "lookout", "hello": "world"},
			identityLabel: map[string]string{"app": "binoculars"},
			portConfig: v1alpha1.PortConfig{
				GrpcPort:    50059,
				HttpPort:    8080,
				MetricsPort: 9000,
			},
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
		},
		"Node port values are translated to ServicePort": {
			name:          "lookout",
			namespace:     "lookout",
			labels:        map[string]string{"app": "lookout", "hello": "world"},
			identityLabel: map[string]string{"app": "binoculars"},
			portConfig: v1alpha1.PortConfig{
				GrpcPort:     50059,
				GrpcNodePort: 32000,
				HttpPort:     8080,
				HttpNodePort: 32001,
				MetricsPort:  9000,
			},
			ports: []corev1.ServicePort{
				{
					Name:     "grpc",
					Port:     50059,
					NodePort: 32000,
				},
				{
					Name:     "web",
					Port:     8080,
					NodePort: 32001,
				},
				{
					Name: "metrics",
					Port: 9000,
				},
			},
		},
	}

	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			got := Service(tc.name, tc.namespace, tc.labels, tc.identityLabel, tc.portConfig)
			assert.Equal(t, tc.name, got.Name)
			assert.Equal(t, tc.namespace, got.Namespace)
			assert.ElementsMatch(t, tc.ports, got.Spec.Ports)
			if tc.portConfig.GrpcNodePort > 0 {
				assert.Equal(t, corev1.ServiceType("NodePort"), got.Spec.Type)
			} else {
				assert.Equal(t, corev1.ServiceType(""), got.Spec.Type)
			}

		})
	}
}
