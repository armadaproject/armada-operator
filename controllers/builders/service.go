package builders

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Service(name string, namespace string, labels, identityLabel map[string]string, ports []corev1.ServicePort) *corev1.Service {
	service := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: labels},
		Spec: corev1.ServiceSpec{
			Selector: identityLabel,
			Ports:    ports,
		},
	}
	return &service
}
