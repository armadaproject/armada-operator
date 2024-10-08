package builders

import (
	"github.com/pkg/errors"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func Ingress(
	name, namespace string,
	labels, annotations map[string]string,
	hostnames []string,
	serviceName, secret, path string,
	servicePort int32,
) (*networkingv1.Ingress, error) {
	if len(hostnames) == 0 {
		// if no hostnames are provided, no ingress can be configured
		return nil, errors.New("no hostnames provided")
	}
	if servicePort <= 0 {
		return nil, errors.New("port must be greater than 0")
	}
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Labels:      labels,
			Annotations: annotations,
		},
	}

	ingress.Spec.TLS = []networkingv1.IngressTLS{{Hosts: hostnames, SecretName: secret}}
	var ingressRules []networkingv1.IngressRule
	for _, val := range hostnames {
		backend := networkingv1.IngressBackend{
			Service: &networkingv1.IngressServiceBackend{
				Name: serviceName,
				Port: networkingv1.ServiceBackendPort{
					Number: servicePort,
				},
			},
		}
		rule := networkingv1.IngressRule{Host: val, IngressRuleValue: networkingv1.IngressRuleValue{
			HTTP: &networkingv1.HTTPIngressRuleValue{
				Paths: []networkingv1.HTTPIngressPath{{
					Path:     path,
					PathType: ptr.To(networkingv1.PathTypePrefix),
					Backend:  backend,
				}},
			},
		}}
		ingressRules = append(ingressRules, rule)
	}
	ingress.Spec.Rules = ingressRules

	return ingress, nil
}
