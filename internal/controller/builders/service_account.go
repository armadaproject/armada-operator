package builders

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	installv1alpha1 "github.com/armadaproject/armada-operator/api/install/v1alpha1"
)

func ServiceAccount(name, namespace string, labels map[string]string, serviceAccountConfig *installv1alpha1.ServiceAccountConfig) *corev1.ServiceAccount {
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: labels},
	}
	if serviceAccountConfig != nil {
		serviceAccount.AutomountServiceAccountToken = serviceAccountConfig.AutomountServiceAccountToken
		serviceAccount.Secrets = serviceAccountConfig.Secrets
		serviceAccount.ImagePullSecrets = serviceAccountConfig.ImagePullSecrets
	}
	return serviceAccount
}
