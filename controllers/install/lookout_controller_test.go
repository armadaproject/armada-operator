package install

import (
	"context"
	"testing"
	"time"

	"github.com/armadaproject/armada-operator/test/k8sclient"

	"github.com/golang/mock/gomock"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/armadaproject/armada-operator/apis/install/v1alpha1"
	installv1alpha1 "github.com/armadaproject/armada-operator/apis/install/v1alpha1"
)

func TestLookoutReconciler_Reconcile(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	scheme, err := installv1alpha1.SchemeBuilder.Build()
	if err != nil {
		t.Fatalf("should not return error when building schema")
	}

	expectedNamespacedName := types.NamespacedName{Namespace: "default", Name: "lookout"}
	expectedClusterResourceNamespacedName := types.NamespacedName{Namespace: "", Name: "lookout"}
	expectedLookout := v1alpha1.Lookout{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Lookout",
			APIVersion: "install.armadaproject.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "lookout"},
		Spec: v1alpha1.LookoutSpec{
			Labels: nil,
			Image: installv1alpha1.Image{
				Repository: "testrepo",
				Tag:        "1.0.0",
			},
			ApplicationConfig: runtime.RawExtension{},
		},
	}

	lookout, err := generateLookoutInstallComponents(&expectedLookout, scheme)
	if err != nil {
		t.Fatal("We should not fail on generating lookout")
	}

	mockK8sClient := k8sclient.NewMockClient(mockCtrl)
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&v1alpha1.Lookout{})).
		Return(nil).
		SetArg(2, expectedLookout)
	// Lookout finalizer
	mockK8sClient.EXPECT().Update(gomock.Any(), gomock.AssignableToTypeOf(&installv1alpha1.Lookout{})).Return(nil)

	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&corev1.Secret{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "lookout"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&corev1.Secret{})).
		Return(nil).
		SetArg(1, *lookout.Secret)

	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&v1.Deployment{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "lookout"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&v1.Deployment{})).
		Return(nil).
		SetArg(1, *lookout.Deployment)

	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&corev1.Service{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "lookout"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&corev1.Service{})).
		Return(nil).
		SetArg(1, *lookout.Service)

	// ClusterRole
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedClusterResourceNamespacedName, gomock.AssignableToTypeOf(&rbacv1.ClusterRole{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "lookout"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&rbacv1.ClusterRole{})).
		Return(nil).
		SetArg(1, *lookout.ClusterRole)
	// ClusterRoleBinding
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedClusterResourceNamespacedName, gomock.AssignableToTypeOf(&rbacv1.ClusterRoleBinding{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "lookout"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&rbacv1.ClusterRoleBinding{})).
		Return(nil).
		SetArg(1, *lookout.ClusterRoleBinding)

	r := LookoutReconciler{
		Client: mockK8sClient,
		Scheme: scheme,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{Namespace: "default", Name: "lookout"},
	}

	_, err = r.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatalf("reconcile should not return error")
	}
}

func TestLookoutReconciler_ReconcileNoLookout(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	expectedNamespacedName := types.NamespacedName{Namespace: "default", Name: "lookout-test"}
	mockK8sClient := k8sclient.NewMockClient(mockCtrl)
	// Executor
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&installv1alpha1.Lookout{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "lookout"))
	scheme, err := installv1alpha1.SchemeBuilder.Build()
	if err != nil {
		t.Fatalf("should not return error when building schema")
	}

	r := LookoutReconciler{
		Client: mockK8sClient,
		Scheme: scheme,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{Namespace: "default", Name: "lookout-test"},
	}

	_, err = r.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatalf("reconcile should not return error")
	}
}
func TestLookoutReconciler_ReconcileDeletingLookout(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	expectedNamespacedName := types.NamespacedName{Namespace: "default", Name: "lookout"}
	expectedLookout := installv1alpha1.Lookout{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Lookout",
			APIVersion: "install.armadaproject.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         "default",
			Name:              "lookout",
			DeletionTimestamp: &metav1.Time{Time: time.Now()},
			Finalizers:        []string{operatorFinalizer},
		},
		Spec: installv1alpha1.LookoutSpec{
			Labels: nil,
			Image: installv1alpha1.Image{
				Repository: "testrepo",
				Tag:        "1.0.0",
			},
			ApplicationConfig: runtime.RawExtension{},
		},
	}
	mockK8sClient := k8sclient.NewMockClient(mockCtrl)
	// Lookout
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&installv1alpha1.Lookout{})).
		Return(nil).
		SetArg(2, expectedLookout)
	// Cleanup
	mockK8sClient.
		EXPECT().
		Delete(gomock.Any(), gomock.AssignableToTypeOf(&rbacv1.ClusterRole{})).
		Return(nil)
	mockK8sClient.
		EXPECT().
		Delete(gomock.Any(), gomock.AssignableToTypeOf(&rbacv1.ClusterRoleBinding{})).
		Return(nil)
	// Remove Lookout Finalizer
	mockK8sClient.
		EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(&installv1alpha1.Lookout{})).
		Return(nil)

	scheme, err := installv1alpha1.SchemeBuilder.Build()
	if err != nil {
		t.Fatalf("should not return error when building schema")
	}

	r := LookoutReconciler{
		Client: mockK8sClient,
		Scheme: scheme,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{Namespace: "default", Name: "lookout"},
	}

	_, err = r.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatalf("reconcile should not return error")
	}
}
