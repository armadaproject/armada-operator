package builders

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func CreateSecret(appConfig runtime.RawExtension, secretName, secretNamespace string) (*corev1.Secret, error) {
	armadaConfig, err := GenerateArmadaConfig(appConfig)
	if err != nil {
		return nil, err
	}
	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: secretNamespace},
		Data:       armadaConfig,
	}
	return &secret, nil
}
