package install

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/armadaproject/armada-operator/test/k8sclient"

	"github.com/golang/mock/gomock"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
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

func TestBinoculars_GenerateBinocularsInstallComponents(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	scheme, err := installv1alpha1.SchemeBuilder.Build()
	if err != nil {
		t.Fatalf("should not return error when building schema")
	}
	cases := map[string]struct {
		binoculars    *v1alpha1.Binoculars
		expectedError bool
	}{
		"binoculars without any errors": {
			binoculars: &installv1alpha1.Binoculars{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Binoculars",
					APIVersion: "install.armadaproject.io/v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "binoculars"},
				Spec: v1alpha1.BinocularsSpec{
					CommonSpecBase: installv1alpha1.CommonSpecBase{
						Labels: map[string]string{"test": "hello"},
						Image: installv1alpha1.Image{
							Repository: "testrepo",
							Tag:        "1.0.0",
						},
						ApplicationConfig: runtime.RawExtension{},
						Resources:         &corev1.ResourceRequirements{},
					},
					Replicas:      2,
					HostNames:     []string{"localhost"},
					ClusterIssuer: "test",
					Ingress: &installv1alpha1.IngressConfig{
						IngressClass: "nginx",
						Labels:       map[string]string{"test": "hello"},
						Annotations:  map[string]string{"test": "hello"},
					},
				},
			},
		},
		"binoculars with invalid config": {
			expectedError: true,
			binoculars: &installv1alpha1.Binoculars{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Binoculars",
					APIVersion: "install.armadaproject.io/v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "binoculars"},
				Spec: v1alpha1.BinocularsSpec{
					CommonSpecBase: installv1alpha1.CommonSpecBase{
						Labels: map[string]string{"test": "hello"},
						Image: installv1alpha1.Image{
							Repository: "testrepo",
							Tag:        "1.0.0",
						},
						ApplicationConfig: runtime.RawExtension{Raw: []byte(`{ "foo": "bar" `)},
						Resources:         &corev1.ResourceRequirements{},
					},
					Replicas:      2,
					HostNames:     []string{"localhost"},
					ClusterIssuer: "test",
					Ingress: &installv1alpha1.IngressConfig{
						IngressClass: "nginx",
						Labels:       map[string]string{"test": "hello"},
						Annotations:  map[string]string{"test": "hello"},
					},
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := generateBinocularsInstallComponents(tc.binoculars, scheme)
			if tc.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBinocularsReconciler_Reconcile(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	scheme, err := installv1alpha1.SchemeBuilder.Build()
	if err != nil {
		t.Fatalf("should not return error when building schema")
	}

	expectedNamespacedName := types.NamespacedName{Namespace: "default", Name: "binoculars"}
	expectedClusterResourceNamespacedName := types.NamespacedName{Namespace: "", Name: "binoculars"}
	expectedBinoculars := v1alpha1.Binoculars{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Binoculars",
			APIVersion: "install.armadaproject.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "binoculars"},
		Spec: v1alpha1.BinocularsSpec{
			CommonSpecBase: installv1alpha1.CommonSpecBase{
				Labels: map[string]string{"test": "hello"},
				Image: installv1alpha1.Image{
					Repository: "testrepo",
					Tag:        "1.0.0",
				},
				ApplicationConfig: runtime.RawExtension{},
				Resources:         &corev1.ResourceRequirements{},
			},

			Replicas:      2,
			HostNames:     []string{"localhost"},
			ClusterIssuer: "test",
			Ingress: &installv1alpha1.IngressConfig{
				IngressClass: "nginx",
				Labels:       map[string]string{"test": "hello"},
				Annotations:  map[string]string{"test": "hello"},
			},
		},
	}

	binoculars, err := generateBinocularsInstallComponents(&expectedBinoculars, scheme)
	if err != nil {
		t.Fatal("We should not fail on generating binoculars")
	}

	mockK8sClient := k8sclient.NewMockClient(mockCtrl)
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&v1alpha1.Binoculars{})).
		Return(nil).
		SetArg(2, expectedBinoculars)
	// Binoculars finalizer
	mockK8sClient.EXPECT().Update(gomock.Any(), gomock.AssignableToTypeOf(&installv1alpha1.Binoculars{})).Return(nil)

	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&corev1.Secret{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "binoculars"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&corev1.Secret{})).
		Return(nil).
		SetArg(1, *binoculars.Secret)

	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&v1.Deployment{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "binoculars"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&v1.Deployment{})).
		Return(nil).
		SetArg(1, *binoculars.Deployment)

	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&corev1.Service{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "binoculars"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&corev1.Service{})).
		Return(nil).
		SetArg(1, *binoculars.Service)

	// ClusterRole
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedClusterResourceNamespacedName, gomock.AssignableToTypeOf(&rbacv1.ClusterRole{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "binoculars"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&rbacv1.ClusterRole{})).
		Return(nil).
		SetArg(1, *binoculars.ClusterRole)
	// ClusterRoleBinding
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedClusterResourceNamespacedName, gomock.AssignableToTypeOf(&rbacv1.ClusterRoleBinding{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "binoculars"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&rbacv1.ClusterRoleBinding{})).
		Return(nil).
		SetArg(1, *binoculars.ClusterRoleBindings[0])
	// Ingress
	ingressNamespacedName := types.NamespacedName{Namespace: "default", Name: "binoculars-rest"}
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), ingressNamespacedName, gomock.AssignableToTypeOf(&networkingv1.Ingress{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "binoculars"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&networkingv1.Ingress{})).
		Return(nil).
		SetArg(1, *binoculars.IngressGrpc)
	ingressGRPCNamespaceNamed := types.NamespacedName{Namespace: "default", Name: "binoculars-grpc"}
	// IngressHttp
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), ingressGRPCNamespaceNamed, gomock.AssignableToTypeOf(&networkingv1.Ingress{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "binoculars"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&networkingv1.Ingress{})).
		Return(nil).SetArg(1, *binoculars.IngressHttp)
	// ServiceAccount
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&corev1.ServiceAccount{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "armadaserver"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&corev1.ServiceAccount{})).
		Return(nil)

	r := BinocularsReconciler{
		Client: mockK8sClient,
		Scheme: scheme,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{Namespace: "default", Name: "binoculars"},
	}

	_, err = r.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatalf("reconcile should not return error")
	}
}

func TestBinocularsReconciler_ReconcileNoBinoculars(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	expectedNamespacedName := types.NamespacedName{Namespace: "default", Name: "binoculars-test"}
	mockK8sClient := k8sclient.NewMockClient(mockCtrl)
	// Executor
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&installv1alpha1.Binoculars{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "binoculars"))
	scheme, err := installv1alpha1.SchemeBuilder.Build()
	if err != nil {
		t.Fatalf("should not return error when building schema")
	}

	r := BinocularsReconciler{
		Client: mockK8sClient,
		Scheme: scheme,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{Namespace: "default", Name: "binoculars-test"},
	}

	_, err = r.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatalf("reconcile should not return error")
	}
}

func TestBinocularsReconciler_ReconcileDeletingBinoculars(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	expectedNamespacedName := types.NamespacedName{Namespace: "default", Name: "binoculars"}
	expectedBinoculars := installv1alpha1.Binoculars{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Binoculars",
			APIVersion: "install.armadaproject.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         "default",
			Name:              "binoculars",
			DeletionTimestamp: &metav1.Time{Time: time.Now()},
			Finalizers:        []string{operatorFinalizer},
		},
		Spec: installv1alpha1.BinocularsSpec{
			HostNames: []string{"ingress.host"},
			CommonSpecBase: installv1alpha1.CommonSpecBase{
				Labels: nil,
				Image: installv1alpha1.Image{
					Repository: "testrepo",
					Tag:        "1.0.0",
				},
				ApplicationConfig: runtime.RawExtension{},
			},
			Replicas:      2,
			ClusterIssuer: "test",
			Ingress: &installv1alpha1.IngressConfig{
				IngressClass: "nginx",
			},
		},
	}
	mockK8sClient := k8sclient.NewMockClient(mockCtrl)
	// Binoculars
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&installv1alpha1.Binoculars{})).
		Return(nil).
		SetArg(2, expectedBinoculars)
	// Cleanup
	mockK8sClient.
		EXPECT().
		Delete(gomock.Any(), gomock.AssignableToTypeOf(&rbacv1.ClusterRole{})).
		Return(nil)
	mockK8sClient.
		EXPECT().
		Delete(gomock.Any(), gomock.AssignableToTypeOf(&rbacv1.ClusterRoleBinding{})).
		Return(nil)
	// Remove Binoculars Finalizer
	mockK8sClient.
		EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(&installv1alpha1.Binoculars{})).
		Return(nil)

	scheme, err := installv1alpha1.SchemeBuilder.Build()
	if err != nil {
		t.Fatalf("should not return error when building schema")
	}

	r := BinocularsReconciler{
		Client: mockK8sClient,
		Scheme: scheme,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{Namespace: "default", Name: "binoculars"},
	}

	_, err = r.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatalf("reconcile should not return error")
	}
}

func TestBinocularsReconciler_ReconcileInvalidApplicationConfig(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	expectedNamespacedName := types.NamespacedName{Namespace: "default", Name: "binoculars"}
	expectedBinoculars := installv1alpha1.Binoculars{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Binoculars",
			APIVersion: "install.armadaproject.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         "default",
			Name:              "binoculars",
			DeletionTimestamp: &metav1.Time{Time: time.Now()},
			Finalizers:        []string{operatorFinalizer},
		},
		Spec: installv1alpha1.BinocularsSpec{
			CommonSpecBase: installv1alpha1.CommonSpecBase{
				Labels: nil,
				Image: installv1alpha1.Image{
					Repository: "testrepo",
					Tag:        "1.0.0",
				},
				ApplicationConfig: runtime.RawExtension{Raw: []byte(`{ "foo": "bar" `)},
			},
			Replicas:      2,
			ClusterIssuer: "test",
			Ingress: &installv1alpha1.IngressConfig{
				IngressClass: "nginx",
			},
		},
	}
	mockK8sClient := k8sclient.NewMockClient(mockCtrl)
	// Binoculars
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&installv1alpha1.Binoculars{})).
		Return(nil).
		SetArg(2, expectedBinoculars)

	scheme, err := installv1alpha1.SchemeBuilder.Build()
	if err != nil {
		t.Fatalf("should not return error when building schema")
	}

	r := BinocularsReconciler{
		Client: mockK8sClient,
		Scheme: scheme,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{Namespace: "default", Name: "binoculars"},
	}

	_, err = r.Reconcile(context.Background(), req)
	assert.Error(t, err)
}
