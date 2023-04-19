package builders

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/armadaproject/armada-operator/apis/install/v1alpha1"
)

func Service(name string, namespace string, labels, identityLabel map[string]string, portConfig v1alpha1.PortConfig) *corev1.Service {
	ports := []corev1.ServicePort{
		{
			Name:     "web",
			Port:     portConfig.HttpPort,
			NodePort: portConfig.HttpNodePort,
		},
		{
			Name:     "grpc",
			Port:     portConfig.GrpcPort,
			NodePort: portConfig.GrpcNodePort,
		},
		{
			Name: "metrics",
			Port: portConfig.MetricsPort,
		},
	}
	service := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: labels},
		Spec: corev1.ServiceSpec{
			Selector: identityLabel,
			Ports:    ports,
		},
	}
	if portConfig.HttpNodePort > 0 || portConfig.GrpcNodePort > 0 {
		service.Spec.Type = "NodePort"
	}

	return &service
}
