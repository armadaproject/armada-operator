package builders

import (
	"testing"

	"k8s.io/utils/ptr"

	"github.com/stretchr/testify/assert"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIngress(t *testing.T) {
	tests := []struct {
		test        string
		labels      map[string]string
		annotations map[string]string
		hostnames   []string
		service     string
		secret      string
		path        string
		port        int32
		expectedErr bool
		expected    *networkingv1.Ingress
	}{
		{
			test:        "valid ingress",
			labels:      map[string]string{"app": "my-app"},
			annotations: map[string]string{"ingress.kubernetes.io/rewrite-target": "/"},
			hostnames:   []string{"example.com"},
			service:     "my-service",
			secret:      "my-secret",
			path:        "/",
			port:        80,
			expectedErr: false,
			expected: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-ingress",
					Namespace: "default",
					Labels:    map[string]string{"app": "my-app"},
					Annotations: map[string]string{
						"ingress.kubernetes.io/rewrite-target": "/",
					},
				},
				Spec: networkingv1.IngressSpec{
					TLS: []networkingv1.IngressTLS{
						{Hosts: []string{"example.com"}, SecretName: "my-secret"},
					},
					Rules: []networkingv1.IngressRule{
						{
							Host: "example.com",
							IngressRuleValue: networkingv1.IngressRuleValue{
								HTTP: &networkingv1.HTTPIngressRuleValue{
									Paths: []networkingv1.HTTPIngressPath{
										{
											Path:     "/",
											PathType: ptr.To(networkingv1.PathTypePrefix),
											Backend: networkingv1.IngressBackend{
												Service: &networkingv1.IngressServiceBackend{
													Name: "my-service",
													Port: networkingv1.ServiceBackendPort{
														Number: 80,
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			test:        "no hostnames",
			labels:      map[string]string{"app": "my-app"},
			annotations: map[string]string{"ingress.kubernetes.io/rewrite-target": "/"},
			hostnames:   []string{},
			service:     "my-service",
			secret:      "my-secret",
			path:        "/",
			port:        80,
			expectedErr: true,
			expected:    nil, // No hostnames, so should return nil
		},
		{
			test:        "multiple hostnames",
			labels:      map[string]string{"app": "my-app"},
			annotations: map[string]string{"ingress.kubernetes.io/rewrite-target": "/"},
			hostnames:   []string{"example.com", "example.org"},
			service:     "my-service",
			secret:      "my-secret",
			path:        "/",
			port:        443,
			expectedErr: false,
			expected: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-ingress",
					Namespace: "default",
					Labels:    map[string]string{"app": "my-app"},
					Annotations: map[string]string{
						"ingress.kubernetes.io/rewrite-target": "/",
					},
				},
				Spec: networkingv1.IngressSpec{
					TLS: []networkingv1.IngressTLS{
						{Hosts: []string{"example.com", "example.org"}, SecretName: "my-secret"},
					},
					Rules: []networkingv1.IngressRule{
						{
							Host: "example.com",
							IngressRuleValue: networkingv1.IngressRuleValue{
								HTTP: &networkingv1.HTTPIngressRuleValue{
									Paths: []networkingv1.HTTPIngressPath{
										{
											Path:     "/",
											PathType: ptr.To(networkingv1.PathTypePrefix),
											Backend: networkingv1.IngressBackend{
												Service: &networkingv1.IngressServiceBackend{
													Name: "my-service",
													Port: networkingv1.ServiceBackendPort{
														Number: 443,
													},
												},
											},
										},
									},
								},
							},
						},
						{
							Host: "example.org",
							IngressRuleValue: networkingv1.IngressRuleValue{
								HTTP: &networkingv1.HTTPIngressRuleValue{
									Paths: []networkingv1.HTTPIngressPath{
										{
											Path:     "/",
											PathType: ptr.To(networkingv1.PathTypePrefix),
											Backend: networkingv1.IngressBackend{
												Service: &networkingv1.IngressServiceBackend{
													Name: "my-service",
													Port: networkingv1.ServiceBackendPort{
														Number: 443,
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.test, func(t *testing.T) {
			ingress, err := Ingress(
				"test-ingress",
				"default",
				tt.labels,
				tt.annotations,
				tt.hostnames,
				tt.service,
				tt.secret,
				tt.path,
				tt.port,
			)
			if tt.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.EqualValues(t, tt.expected, ingress)
		})
	}
}
