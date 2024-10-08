package builders

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestService(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		name          string
		namespace     string
		labels        map[string]string
		identityLabel map[string]string
		config        CommonApplicationConfig
		portConfig    PortConfig
		expectedPorts []corev1.ServicePort
	}{
		"All ports generated correct": {
			name:          "lookout",
			namespace:     "lookout",
			labels:        map[string]string{"app": "lookout", "hello": "world"},
			identityLabel: map[string]string{"app": "binoculars"},
			config: CommonApplicationConfig{
				GRPCPort:    50059,
				HTTPPort:    8080,
				MetricsPort: 9000,
			},
			portConfig: PortConfig{
				ExposeHTTP:    true,
				ExposeGRPC:    true,
				ExposeMetrics: true,
			},
			expectedPorts: []corev1.ServicePort{
				{
					Name:     "grpc",
					Port:     50059,
					Protocol: corev1.ProtocolTCP,
				},
				{
					Name:     "web",
					Port:     8080,
					Protocol: corev1.ProtocolTCP,
				},
				{
					Name:     "metrics",
					Port:     9000,
					Protocol: corev1.ProtocolTCP,
				},
			},
		},
		"Node port values are translated to ServicePort": {
			name:          "lookout",
			namespace:     "lookout",
			labels:        map[string]string{"app": "lookout", "hello": "world"},
			identityLabel: map[string]string{"app": "binoculars"},
			config: CommonApplicationConfig{
				GRPCPort:     50059,
				GRPCNodePort: 32000,
				HTTPPort:     8080,
				HTTPNodePort: 32001,
				MetricsPort:  9000,
				Profiling: ProfilingConfig{
					Port: 1337,
				},
			},
			portConfig: PortConfig{
				ExposeHTTP:      true,
				ExposeGRPC:      true,
				ExposeMetrics:   true,
				ExposeProfiling: true,
			},
			expectedPorts: []corev1.ServicePort{
				{
					Name:     "grpc",
					Port:     50059,
					NodePort: 32000,
					Protocol: corev1.ProtocolTCP,
				},
				{
					Name:     "web",
					Port:     8080,
					NodePort: 32001,
					Protocol: corev1.ProtocolTCP,
				},
				{
					Name:     "metrics",
					Port:     9000,
					Protocol: corev1.ProtocolTCP,
				},
				{
					Name:     "profiling",
					Port:     1337,
					Protocol: corev1.ProtocolTCP,
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := Service(tc.name, tc.namespace, tc.labels, tc.identityLabel, &tc.config, tc.portConfig)
			assert.Equal(t, tc.name, got.Name)
			assert.Equal(t, tc.namespace, got.Namespace)
			assert.ElementsMatch(t, tc.expectedPorts, got.Spec.Ports)
			if tc.config.GRPCNodePort > 0 || tc.config.HTTPNodePort > 0 {
				assert.Equal(t, corev1.ServiceTypeNodePort, got.Spec.Type)
			} else {
				assert.Equal(t, corev1.ServiceTypeClusterIP, got.Spec.Type)
			}
		})
	}
}
