package builders

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"

	installv1alpha1 "github.com/armadaproject/armada-operator/api/install/v1alpha1"
)

func Test_ServiceAccount(t *testing.T) {

	tests := map[string]struct {
		name                 string
		namespace            string
		labels               map[string]string
		serviceAccountConfig *installv1alpha1.ServiceAccountConfig
	}{
		"it builds service account from name and namespace": {
			name:      "test",
			namespace: "default",
		},
		"it builds service account with all values specificed": {
			name:                 "test",
			namespace:            "default",
			labels:               map[string]string{"hello": "world"},
			serviceAccountConfig: &installv1alpha1.ServiceAccountConfig{Secrets: []v1.ObjectReference{}, AutomountServiceAccountToken: pointer.Bool(true), ImagePullSecrets: []v1.LocalObjectReference{}},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			output := CreateServiceAccount(tt.name, tt.namespace, tt.labels, tt.serviceAccountConfig)
			assert.Equal(t, "test", output.Name)
			assert.Equal(t, "default", output.Namespace)
			assert.Equal(t, tt.labels, output.Labels)
			if tt.serviceAccountConfig != nil {
				assert.Equal(t, tt.serviceAccountConfig.AutomountServiceAccountToken, output.AutomountServiceAccountToken)
				assert.Equal(t, tt.serviceAccountConfig.ImagePullSecrets, output.ImagePullSecrets)
				assert.Equal(t, tt.serviceAccountConfig.Secrets, output.Secrets)
			} else {
				assert.Nil(t, tt.serviceAccountConfig)
			}
		})
	}
}
