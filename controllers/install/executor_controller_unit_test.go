package install

import (
	"context"
	"testing"

	"github.com/armadaproject/armada-operator/apis/install/v1alpha1"
	"github.com/armadaproject/armada-operator/internal/k8sclient"
	"github.com/golang/mock/gomock"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestExecutorReconciler_Reconcile(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	expectedNamespacedName := types.NamespacedName{Namespace: "default", Name: "executor"}
	expectedExecutor := v1alpha1.Executor{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Executor",
			APIVersion: "install.armadaproject.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "executor"},
		Spec: v1alpha1.ExecutorSpec{
			Name:   "executor",
			Labels: nil,
			Image: v1alpha1.Image{
				Repository: "testrepo",
				Image:      "executor",
				Tag:        "1.0.0",
			},
			ApplicationConfig: map[string]runtime.RawExtension{},
		},
	}
	mockK8sClient := k8sclient.NewMockClient(mockCtrl)
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&v1alpha1.Executor{})).
		Return(nil).
		SetArg(2, expectedExecutor)
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&rbacv1.ClusterRole{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "executor"))
	expectedClusterRole := rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "executor"},
		Rules: []rbacv1.PolicyRule{{
			Verbs:     []string{"create"},
			APIGroups: []string{""},
			Resources: []string{"pods"},
		}},
	}
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(gomock.AssignableToTypeOf(&rbacv1.ClusterRole{}))).
		Return(nil).
		SetArg(2, expectedClusterRole)
	scheme, err := v1alpha1.SchemeBuilder.Build()
	if err != nil {
		t.Fatalf("should not return error when building schema")
	}

	r := ExecutorReconciler{
		Client: mockK8sClient,
		Scheme: scheme,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{Namespace: "default", Name: "executor"},
	}

	_, err = r.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatalf("reconcile should not return error")
	}
}
