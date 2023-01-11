package install

import (
	"context"
	"testing"

	"github.com/armadaproject/armada-operator/test/k8sclient"

	installv1alpha1 "github.com/armadaproject/armada-operator/apis/install/v1alpha1"
	"github.com/golang/mock/gomock"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestExecutorReconciler_ReconcileNewExecutor(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	scheme, err := installv1alpha1.SchemeBuilder.Build()
	if err != nil {
		t.Fatalf("should not return error when building schema")
	}

	expectedNamespacedName := types.NamespacedName{Namespace: "default", Name: "executor"}
	expectedExecutor := installv1alpha1.Executor{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Executor",
			APIVersion: "install.armadaproject.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "executor"},
		Spec: installv1alpha1.ExecutorSpec{
			Labels: nil,
			Image: installv1alpha1.Image{
				Repository: "testrepo",
				Tag:        "1.0.0",
			},
			ApplicationConfig: runtime.RawExtension{},
		},
	}

	executorComponents, err := generateExecutorInstallComponents(&expectedExecutor, scheme)
	if err != nil {
		t.Errorf("Executor Install Components should not fail")
	}

	mockK8sClient := k8sclient.NewMockClient(mockCtrl)
	// Executor
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&installv1alpha1.Executor{})).
		Return(nil).
		SetArg(2, expectedExecutor)
	// Executor finalizer
	mockK8sClient.EXPECT().Update(gomock.Any(), gomock.AssignableToTypeOf(&installv1alpha1.Executor{})).Return(nil)
	// ServiceAccount
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&corev1.ServiceAccount{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "executor"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&corev1.ServiceAccount{})).
		Return(nil)
	// Secret
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&corev1.Secret{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "executor"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&corev1.Secret{})).
		Return(nil).
		SetArg(1, *executorComponents.Secret)
	// Deployment
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&appsv1.Deployment{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "executor"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&appsv1.Deployment{})).
		Return(nil).
		SetArg(1, *executorComponents.Deployment)
	// Deployment
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&corev1.Service{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "executor"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&corev1.Service{})).
		Return(nil).
		SetArg(1, *executorComponents.Service)

	// // ClusterRole
	// mockK8sClient.
	// 	EXPECT().
	// 	Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&rbacv1.ClusterRole{})).
	// 	Return(errors.NewNotFound(schema.GroupResource{}, "executor"))
	// mockK8sClient.
	// 	EXPECT().
	// 	Create(gomock.Any(), gomock.AssignableToTypeOf(&rbacv1.ClusterRole{})).
	// 	Return(nil).
	// 	SetArg(1, *executorComponents.ClusterRole)
	// // // ClusterRoleBinding
	// // mockK8sClient.
	// // 	EXPECT().
	// // 	Get(gomock.Any(), "executor", gomock.AssignableToTypeOf(&rbacv1.ClusterRoleBinding{})).
	// // 	Return(errors.NewNotFound(schema.GroupResource{}, "executor"))
	// // mockK8sClient.
	// // 	EXPECT().
	// // 	Create(gomock.Any(), gomock.AssignableToTypeOf(&rbacv1.ClusterRoleBinding{})).
	// // 	Return(nil).
	// // 	SetArg(1, *executorComponents.ClusterRoleBinding)

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

func TestExecutorReconciler_ReconcileNoExecutor(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	expectedNamespacedName := types.NamespacedName{Namespace: "default", Name: "executor-test"}
	mockK8sClient := k8sclient.NewMockClient(mockCtrl)
	// Executor
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&installv1alpha1.Executor{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "executor"))
	scheme, err := installv1alpha1.SchemeBuilder.Build()
	if err != nil {
		t.Fatalf("should not return error when building schema")
	}

	r := ExecutorReconciler{
		Client: mockK8sClient,
		Scheme: scheme,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{Namespace: "default", Name: "executor-test"},
	}

	_, err = r.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatalf("reconcile should not return error")
	}
}
