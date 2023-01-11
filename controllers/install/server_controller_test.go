package install

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/armadaproject/armada-operator/apis/install/v1alpha1"
	"github.com/armadaproject/armada-operator/internal/k8sclient"
)

func TestServerReconciler_Reconcile(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	scheme, err := v1alpha1.SchemeBuilder.Build()
	if err != nil {
		t.Fatalf("should not return error when building schema")
	}
	mockK8sClient := k8sclient.NewMockClient(mockCtrl)

	r := ServerReconciler{
		Client: mockK8sClient,
		Scheme: scheme,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{Namespace: "default", Name: "server"},
	}

	_, err = r.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatalf("reconcile should not return error")
	}
}
