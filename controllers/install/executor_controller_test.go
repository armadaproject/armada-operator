package install

import (
	"context"
	"testing"
	"time"

	"github.com/armadaproject/armada-operator/test/k8sclient"

	"github.com/stretchr/testify/assert"

	"github.com/golang/mock/gomock"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	installv1alpha1 "github.com/armadaproject/armada-operator/apis/install/v1alpha1"
)

func TestExecutorReconciler_ReconcileNewExecutor(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	expectedNamespacedName := types.NamespacedName{Namespace: "default", Name: "executor"}
	expectedClusterResourceNamespacedName := types.NamespacedName{Namespace: "", Name: "executor"}
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
			Resources:         &corev1.ResourceRequirements{},
			Prometheus:        &installv1alpha1.PrometheusConfig{Enabled: true, ScrapeInterval: &metav1.Duration{Duration: 1 * time.Second}},
		},
	}
	mockK8sClient := k8sclient.NewMockClient(mockCtrl)
	// Executor
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&installv1alpha1.Executor{})).
		Return(nil).
		SetArg(2, expectedExecutor)
	// Executor finalizer
	mockK8sClient.
		EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(&installv1alpha1.Executor{})).
		Return(nil)
	// ServiceAccount
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&corev1.ServiceAccount{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "executor"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&corev1.ServiceAccount{})).
		Return(nil)
	// ClusterRole
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedClusterResourceNamespacedName, gomock.AssignableToTypeOf(&rbacv1.ClusterRole{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "executor"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&rbacv1.ClusterRole{})).
		Return(nil)
	// ClusterRoleBinding
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedClusterResourceNamespacedName, gomock.AssignableToTypeOf(&rbacv1.ClusterRoleBinding{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "executor"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&rbacv1.ClusterRoleBinding{})).
		Return(nil)
	// Secret
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&corev1.Secret{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "executor"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&corev1.Secret{})).
		Return(nil)
	// Deployment
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&appsv1.Deployment{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "executor"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&appsv1.Deployment{})).
		Return(nil)
	// Service
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&corev1.Service{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "executor"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&corev1.Service{})).
		Return(nil)

	scheme, err := installv1alpha1.SchemeBuilder.Build()
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

func TestExecutorReconciler_ReconcileNoExecutor(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	expectedNamespacedName := types.NamespacedName{Namespace: "default", Name: "executor"}
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
		NamespacedName: types.NamespacedName{Namespace: "default", Name: "executor"},
	}

	_, err = r.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatalf("reconcile should not return error")
	}
}

func TestExecutorReconciler_ReconcileDeletingExecutor(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	expectedNamespacedName := types.NamespacedName{Namespace: "default", Name: "executor"}
	expectedExecutor := installv1alpha1.Executor{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Executor",
			APIVersion: "install.armadaproject.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         "default",
			Name:              "executor",
			DeletionTimestamp: &metav1.Time{Time: time.Now()},
			Finalizers:        []string{operatorFinalizer},
		},
		Spec: installv1alpha1.ExecutorSpec{
			Labels: nil,
			Image: installv1alpha1.Image{
				Repository: "testrepo",
				Tag:        "1.0.0",
			},
			ApplicationConfig: runtime.RawExtension{},
		},
	}
	mockK8sClient := k8sclient.NewMockClient(mockCtrl)
	// Executor
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&installv1alpha1.Executor{})).
		Return(nil).
		SetArg(2, expectedExecutor)
	// Cleanup
	mockK8sClient.
		EXPECT().
		Delete(gomock.Any(), gomock.AssignableToTypeOf(&rbacv1.ClusterRole{})).
		Return(nil)
	mockK8sClient.
		EXPECT().
		Delete(gomock.Any(), gomock.AssignableToTypeOf(&rbacv1.ClusterRoleBinding{})).
		Return(nil)
	// Remove Executor Finalizer
	mockK8sClient.
		EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(&installv1alpha1.Executor{})).
		Return(nil)

	scheme, err := installv1alpha1.SchemeBuilder.Build()
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
func TestExecutorReconciler_ReconcileErrorOnApplicationConfig(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	expectedNamespacedName := types.NamespacedName{Namespace: "default", Name: "executor"}
	expectedExecutor := installv1alpha1.Executor{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Executor",
			APIVersion: "install.armadaproject.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         "default",
			Name:              "executor",
			DeletionTimestamp: &metav1.Time{Time: time.Now()},
			Finalizers:        []string{operatorFinalizer},
		},
		Spec: installv1alpha1.ExecutorSpec{
			Labels: nil,
			Image: installv1alpha1.Image{
				Repository: "testrepo",
				Tag:        "1.0.0",
			},
			ApplicationConfig: runtime.RawExtension{Raw: []byte(`{ "foo": "bar" `)},
		},
	}
	mockK8sClient := k8sclient.NewMockClient(mockCtrl)
	// Executor
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&installv1alpha1.Executor{})).
		Return(nil).
		SetArg(2, expectedExecutor)

	scheme, err := installv1alpha1.SchemeBuilder.Build()
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
	assert.Error(t, err)
}

func TestExecutorReconciler_generateAdditionalClusterRoles(t *testing.T) {
	expectedExecutor := installv1alpha1.Executor{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Executor",
			APIVersion: "install.armadaproject.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test-executor"},
		Spec: installv1alpha1.ExecutorSpec{
			CustomServiceAccount: "test-service-account",
			AdditionalClusterRoleBindings: []installv1alpha1.AdditionalClusterRoleBinding{
				{
					NameSuffix:      "test1",
					ClusterRoleName: "foo",
				},
				{
					NameSuffix:      "test2",
					ClusterRoleName: "bar",
				},
			},
		},
	}
	r := ExecutorReconciler{}
	bindings := r.createAdditionalClusterRoleBindings(&expectedExecutor, "test-service-account")
	expectedClusterRoleBinding1 := rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-executor-test1",
			Namespace: "",
			Labels:    map[string]string{"app": "test-executor", "release": "test-executor"},
		},
		Subjects: []rbacv1.Subject{{
			Kind:      "ServiceAccount",
			Name:      "test-service-account",
			Namespace: "default",
		}},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "foo",
		},
	}
	assert.Equal(t, expectedClusterRoleBinding1, *bindings[0])
	expectedClusterRoleBinding2 := rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-executor-test2",
			Namespace: "",
			Labels:    map[string]string{"app": "test-executor", "release": "test-executor"},
		},
		Subjects: []rbacv1.Subject{{
			Kind:      "ServiceAccount",
			Name:      "test-service-account",
			Namespace: "default",
		}},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "bar",
		},
	}
	assert.Equal(t, expectedClusterRoleBinding2, *bindings[1])
}
