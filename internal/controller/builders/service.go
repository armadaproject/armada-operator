package builders

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PortConfig specifies which ports should be exposed by the service
type PortConfig struct {
	ExposeHTTP      bool
	ExposeGRPC      bool
	ExposeMetrics   bool
	ExposeProfiling bool
}

var ServiceEnableApplicationPortsOnly = PortConfig{
	ExposeHTTP:    true,
	ExposeGRPC:    true,
	ExposeMetrics: true,
}

var ServiceEnableHTTPWithMetrics = PortConfig{
	ExposeHTTP:    true,
	ExposeMetrics: true,
}

var ServiceEnableGRPCWithMetrics = PortConfig{
	ExposeGRPC:    true,
	ExposeMetrics: true,
}

var ServiceEnableProfilingPortOnly = PortConfig{
	ExposeProfiling: true,
}

var ServiceEnableMetricsPortOnly = PortConfig{
	ExposeMetrics: true,
}

func Service(
	name string,
	namespace string,
	labels, identityLabel map[string]string,
	appConfig *CommonApplicationConfig,
	portConfig PortConfig,
) *corev1.Service {
	var ports []corev1.ServicePort
	if portConfig.ExposeHTTP {
		ports = append(ports, corev1.ServicePort{
			Name:     "web",
			Port:     appConfig.HTTPPort,
			NodePort: appConfig.HTTPNodePort,
			Protocol: corev1.ProtocolTCP,
		})
	}
	if portConfig.ExposeGRPC {
		ports = append(ports, corev1.ServicePort{
			Name:     "grpc",
			Port:     appConfig.GRPCPort,
			NodePort: appConfig.GRPCNodePort,
			Protocol: corev1.ProtocolTCP,
		})
	}
	if portConfig.ExposeMetrics {
		ports = append(ports, corev1.ServicePort{
			Name:     "metrics",
			Port:     appConfig.MetricsPort,
			Protocol: corev1.ProtocolTCP,
		})
	}
	if port := appConfig.Profiling.Port; portConfig.ExposeProfiling {
		ports = append(ports, corev1.ServicePort{
			Name:     "profiling",
			Port:     port,
			Protocol: corev1.ProtocolTCP,
		})
	}
	service := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: labels},
		Spec: corev1.ServiceSpec{
			Selector: identityLabel,
			Ports:    ports,
			Type:     corev1.ServiceTypeClusterIP,
		},
	}
	if appConfig.HTTPNodePort > 0 || appConfig.GRPCNodePort > 0 {
		service.Spec.Type = corev1.ServiceTypeNodePort
	}

	return &service
}
