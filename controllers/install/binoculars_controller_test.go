package install

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/armadaproject/armada-operator/test/k8sclient"

	"github.com/golang/mock/gomock"
	appsv1 "k8s.io/api/apps/v1"
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
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/armadaproject/armada-operator/apis/install/v1alpha1"
	installv1alpha1 "github.com/armadaproject/armada-operator/apis/install/v1alpha1"
)

func TestBinoculars_GenerateInstallComponents(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	scheme, err := installv1alpha1.SchemeBuilder.Build()
	if err != nil {
		t.Fatalf("should not return error when building schema")
	}

	validBinoculars := &v1alpha1.Binoculars{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Binoculars",
			APIVersion: "install.armadaproject.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "binoculars"},
		Spec: v1alpha1.BinocularsSpec{
			Replicas:  2,
			HostNames: []string{"localhost"},
			Labels:    map[string]string{"test": "hello"},
			Image: installv1alpha1.Image{
				Repository: "testrepo",
				Tag:        "1.0.0",
			},
			ApplicationConfig: runtime.RawExtension{},
			ClusterIssuer:     "test",
			Ingress: &installv1alpha1.IngressConfig{
				IngressClass: "nginx",
				Labels:       map[string]string{"test": "hello"},
				Annotations:  map[string]string{"test": "hello"},
			},
			Resources: &corev1.ResourceRequirements{},
		},
	}

	tests := map[string]struct {
		binoculars     *installv1alpha1.Binoculars
		ownerReference func(owner metav1.Object, object metav1.Object, scheme *runtime.Scheme) error
		expectedError  bool
	}{
		"it builds binoculars components": {
			ownerReference: func(owner metav1.Object, object metav1.Object, scheme *runtime.Scheme) error { return nil },
			binoculars:     validBinoculars,
		},
		"it generates an error if binoculars config is malformed": {
			expectedError:  true,
			ownerReference: func(owner metav1.Object, object metav1.Object, scheme *runtime.Scheme) error { return nil },
			binoculars: &v1alpha1.Binoculars{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Binoculars",
					APIVersion: "install.armadaproject.io/v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "binoculars"},
				Spec: v1alpha1.BinocularsSpec{
					Replicas:  2,
					HostNames: []string{"localhost"},
					Labels:    map[string]string{"test": "hello"},
					Image: installv1alpha1.Image{
						Repository: "testrepo",
						Tag:        "1.0.0",
					},
					ApplicationConfig: runtime.RawExtension{Raw: []byte(`{ "foo": "bar" `)},
					ClusterIssuer:     "test",
					Ingress: &installv1alpha1.IngressConfig{
						IngressClass: "nginx",
						Labels:       map[string]string{"test": "hello"},
						Annotations:  map[string]string{"test": "hello"},
					},
				},
			},
		},
		"it generates an error if ownerReference has an error on IngressRest SetOwnerReferences": {
			expectedError: true,
			ownerReference: func(owner metav1.Object, object metav1.Object, scheme *runtime.Scheme) error {
				if strings.Contains(object.GetName(), "rest") {
					return fmt.Errorf("mocked error during ingressRest")
				}
				return nil
			},
			binoculars: validBinoculars,
		},
		"it generates an error if ownerReference has an error on IngressGrpc SetOwnerReferences": {
			expectedError: true,
			ownerReference: func(owner metav1.Object, object metav1.Object, scheme *runtime.Scheme) error {
				if strings.Contains(object.GetName(), "grpc") {
					return fmt.Errorf("mocked error during ingressGrpc")
				}
				return nil
			},
			binoculars: validBinoculars,
		},
		"it generates an error if ownerReference has an error on service SetOwnerReferences": {
			expectedError: true,
			ownerReference: func(owner metav1.Object, object metav1.Object, scheme *runtime.Scheme) error {
				_, ok := object.(*corev1.Service)
				if ok {
					return fmt.Errorf("mocked error on service")
				}
				return nil
			},
			binoculars: validBinoculars,
		},
		"it generates an error if ownerReference has an error on secret SetOwnerReferences": {
			expectedError: true,
			ownerReference: func(owner metav1.Object, object metav1.Object, scheme *runtime.Scheme) error {
				_, ok := object.(*corev1.Secret)
				if ok {
					return fmt.Errorf("mocked error on secret")
				}
				return nil
			},
			binoculars: validBinoculars,
		},
		"it generates an error if ownerReference has an error on deployment SetOwnerReferences": {
			expectedError: true,
			ownerReference: func(owner metav1.Object, object metav1.Object, scheme *runtime.Scheme) error {
				_, ok := object.(*appsv1.Deployment)
				if ok {
					return fmt.Errorf("mocked error on deployment")
				}
				return nil
			},
			binoculars: validBinoculars,
		},
		"it generates an error if ownerReference has an error on serviceAccount SetOwnerReferences": {
			expectedError: true,
			ownerReference: func(owner metav1.Object, object metav1.Object, scheme *runtime.Scheme) error {
				_, ok := object.(*corev1.ServiceAccount)
				if ok {
					return fmt.Errorf("mocked error on service account")
				}
				return nil
			},
			binoculars: validBinoculars,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := generateBinocularsInstallComponents(tt.binoculars, scheme, tt.ownerReference)
			if tt.expectedError && err == nil {
				t.Fatalf("Expected Error but did not get one")
			}
		})
	}
}

func TestBinoculars_upsertComponents(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	validComponents := &BinocularsComponents{
		Deployment:         &appsv1.Deployment{},
		ClusterRole:        &rbacv1.ClusterRole{},
		ClusterRoleBinding: &rbacv1.ClusterRoleBinding{},
		IngressRest:        &networkingv1.Ingress{ObjectMeta: metav1.ObjectMeta{Name: "binoculars-rest"}},
		Ingress:            &networkingv1.Ingress{ObjectMeta: metav1.ObjectMeta{Name: "binoculars-grpc"}},
		Service:            &corev1.Service{},
		ServiceAccount:     &corev1.ServiceAccount{},
		Secret:             &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"}},
	}

	tests := map[string]struct {
		bc            *BinocularsComponents
		mockFnc       func(ctx context.Context, c client.Client, obj client.Object, f controllerutil.MutateFn) (controllerutil.OperationResult, error)
		expectedError bool
	}{
		"it builds upserts without any issues components": {
			mockFnc: func(ctx context.Context, c client.Client, obj client.Object, f controllerutil.MutateFn) (controllerutil.OperationResult, error) {
				return "success", nil
			},
			bc: validComponents,
		},
		"it generates an error if mockFnc has an error on IngressRest": {
			expectedError: true,
			mockFnc: func(ctx context.Context, c client.Client, obj client.Object, f controllerutil.MutateFn) (controllerutil.OperationResult, error) {
				if strings.Contains(obj.GetName(), "rest") {
					return "fail", fmt.Errorf("error in rest ingress")
				}
				return "success", nil
			},
			bc: validComponents,
		},
		"it generates an error if mockFnc has an error on IngressGrpc": {
			expectedError: true,
			mockFnc: func(ctx context.Context, c client.Client, obj client.Object, f controllerutil.MutateFn) (controllerutil.OperationResult, error) {
				if strings.Contains(obj.GetName(), "grpc") {
					return "fail", fmt.Errorf("error in rest ingress")
				}
				return "success", nil
			},
			bc: validComponents,
		},
		"it generates an error if mockFnc has an error on service": {
			expectedError: true,
			mockFnc: func(ctx context.Context, c client.Client, obj client.Object, f controllerutil.MutateFn) (controllerutil.OperationResult, error) {
				_, ok := obj.(*corev1.Service)
				if ok {
					return "mockFail", fmt.Errorf("mocked fail on service")
				}
				return "success", nil
			},
			bc: validComponents,
		},
		"it generates an error if mockFnc has an error on secret": {
			expectedError: true,
			mockFnc: func(ctx context.Context, c client.Client, obj client.Object, f controllerutil.MutateFn) (controllerutil.OperationResult, error) {
				_, ok := obj.(*corev1.Secret)
				if ok {
					return "mockFail", fmt.Errorf("mocked fail on secret")
				}
				return "success", nil
			},
			bc: validComponents,
		},
		"it generates an error if mockFnc has an error on deployment": {
			expectedError: true,
			mockFnc: func(ctx context.Context, c client.Client, obj client.Object, f controllerutil.MutateFn) (controllerutil.OperationResult, error) {
				_, ok := obj.(*appsv1.Deployment)
				if ok {
					return "mockFail", fmt.Errorf("mocked fail on deployment")
				}
				return "success", nil
			},
			bc: validComponents,
		},
		"it generates an error if mockFnc has error on serviceAccount": {
			expectedError: true,
			mockFnc: func(ctx context.Context, c client.Client, obj client.Object, f controllerutil.MutateFn) (controllerutil.OperationResult, error) {
				_, ok := obj.(*corev1.ServiceAccount)
				if ok {
					return "mockFail", fmt.Errorf("mocked fail on serviceAccount")
				}
				return "success", nil
			},
			bc: validComponents,
		},
		"it generates an error if mockFnc has error on clusterRole": {
			expectedError: true,
			mockFnc: func(ctx context.Context, c client.Client, obj client.Object, f controllerutil.MutateFn) (controllerutil.OperationResult, error) {
				_, ok := obj.(*rbacv1.ClusterRole)
				if ok {
					return "mockFail", fmt.Errorf("mocked fail on clusterRole")
				}
				return "success", nil
			},
			bc: validComponents,
		},
		"it generates an error if mockFnc has error on clusterRoleBinding": {
			expectedError: true,
			mockFnc: func(ctx context.Context, c client.Client, obj client.Object, f controllerutil.MutateFn) (controllerutil.OperationResult, error) {
				_, ok := obj.(*rbacv1.ClusterRoleBinding)
				if ok {
					return "mockFail", fmt.Errorf("mocked fail on clusterRoleBinding")
				}
				return "success", nil
			},
			bc: validComponents,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockK8sClient := k8sclient.NewMockClient(mockCtrl)

			params := &UpsertParams{Context: context.Background(), BincularsComponents: tt.bc, Client: mockK8sClient, MutaFnc: func() error { return nil }, CreateOrUpdate: tt.mockFnc}
			_, err := upsertComponents(params)
			if tt.expectedError && err == nil {
				t.Fatalf("Expected Error but did not get one")
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
			Replicas:  2,
			HostNames: []string{"localhost"},
			Labels:    map[string]string{"test": "hello"},
			Image: installv1alpha1.Image{
				Repository: "testrepo",
				Tag:        "1.0.0",
			},
			ApplicationConfig: runtime.RawExtension{},
			ClusterIssuer:     "test",
			Ingress: &installv1alpha1.IngressConfig{
				IngressClass: "nginx",
				Labels:       map[string]string{"test": "hello"},
				Annotations:  map[string]string{"test": "hello"},
			},
		},
	}

	binoculars, err := generateBinocularsInstallComponents(&expectedBinoculars, scheme, controllerutil.SetOwnerReference)
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
		SetArg(1, *binoculars.ClusterRoleBinding)
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
		SetArg(1, *binoculars.Ingress)
	ingressGRPCNamespaceNamed := types.NamespacedName{Namespace: "default", Name: "binoculars-grpc"}
	// IngressRest
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), ingressGRPCNamespaceNamed, gomock.AssignableToTypeOf(&networkingv1.Ingress{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "binoculars"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&networkingv1.Ingress{})).
		Return(nil).SetArg(1, *binoculars.IngressRest)
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
			Replicas: 2,
			Labels:   nil,
			Image: installv1alpha1.Image{
				Repository: "testrepo",
				Tag:        "1.0.0",
			},
			ApplicationConfig: runtime.RawExtension{},
			ClusterIssuer:     "test",
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

func TestBinocularsReconciler_ReconcileErrorOnDeletingInUpdate(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	expectedNamespacedName := types.NamespacedName{Namespace: "default", Name: "binoculars"}

	tests := map[string]struct {
		binoculars     *installv1alpha1.Binoculars
		ownerReference func(owner metav1.Object, object metav1.Object, scheme *runtime.Scheme) error
		expectedError  bool
	}{
		"it deletes with a deletion timestamp": {
			ownerReference: func(owner metav1.Object, object metav1.Object, scheme *runtime.Scheme) error { return nil },
			binoculars: &installv1alpha1.Binoculars{
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
					Replicas: 2,
					Labels:   nil,
					Image: installv1alpha1.Image{
						Repository: "testrepo",
						Tag:        "1.0.0",
					},
					ApplicationConfig: runtime.RawExtension{},
					ClusterIssuer:     "test",
					Ingress: &installv1alpha1.IngressConfig{
						IngressClass: "nginx",
					},
				},
			},
		},
		"it detects no deletion timestamp but has a finalizer": {
			ownerReference: func(owner metav1.Object, object metav1.Object, scheme *runtime.Scheme) error { return nil },
			binoculars: &installv1alpha1.Binoculars{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Binoculars",
					APIVersion: "install.armadaproject.io/v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "binoculars",
				},
				Spec: installv1alpha1.BinocularsSpec{
					Replicas: 2,
					Labels:   nil,
					Image: installv1alpha1.Image{
						Repository: "testrepo",
						Tag:        "1.0.0",
					},
					ApplicationConfig: runtime.RawExtension{},
					ClusterIssuer:     "test",
					Ingress: &installv1alpha1.IngressConfig{
						IngressClass: "nginx",
					},
				},
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockK8sClient := k8sclient.NewMockClient(mockCtrl)
			// Binoculars
			mockK8sClient.
				EXPECT().
				Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&installv1alpha1.Binoculars{})).
				Return(nil).
				SetArg(2, *tt.binoculars)
			if tt.binoculars.DeletionTimestamp != nil {
				// Cleanup
				mockK8sClient.
					EXPECT().
					Delete(gomock.Any(), gomock.AssignableToTypeOf(&rbacv1.ClusterRole{})).
					Return(nil)
				mockK8sClient.
					EXPECT().
					Delete(gomock.Any(), gomock.AssignableToTypeOf(&rbacv1.ClusterRoleBinding{})).
					Return(nil)
			}
			// Remove Binoculars Finalizer
			mockK8sClient.
				EXPECT().
				Update(gomock.Any(), gomock.AssignableToTypeOf(&installv1alpha1.Binoculars{})).
				Return(fmt.Errorf("error"))

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
			if err == nil {
				t.Fatalf("reconcile should return error")
			}

		})
	}
}

func TestBinocularsReconciler_ReconcileErrorOnDeletingClusterRoleBinding(t *testing.T) {
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
			Replicas: 2,
			Labels:   nil,
			Image: installv1alpha1.Image{
				Repository: "testrepo",
				Tag:        "1.0.0",
			},
			ApplicationConfig: runtime.RawExtension{},
			ClusterIssuer:     "test",
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
		Return(fmt.Errorf("Error on Deleting ClusterRole"))

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
	if err == nil {
		t.Fatalf("reconcile should return error")
	}
}

func TestBinocularsReconciler_ReconcileErrorOnDeletingClusterRole(t *testing.T) {
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
			Replicas: 2,
			Labels:   nil,
			Image: installv1alpha1.Image{
				Repository: "testrepo",
				Tag:        "1.0.0",
			},
			ApplicationConfig: runtime.RawExtension{},
			ClusterIssuer:     "test",
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
		Return(fmt.Errorf("Error"))

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
	if err == nil {
		t.Fatalf("reconcile should return error")
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
			Namespace: "default",
			Name:      "binoculars",
		},
		Spec: installv1alpha1.BinocularsSpec{
			Replicas: 2,
			Labels:   nil,
			Image: installv1alpha1.Image{
				Repository: "testrepo",
				Tag:        "1.0.0",
			},
			ApplicationConfig: runtime.RawExtension{Raw: []byte(`{ "foo": "bar" `)},
			ClusterIssuer:     "test",
			Ingress: &installv1alpha1.IngressConfig{
				IngressClass: "nginx",
			},
		},
	}
	mockK8sClient := k8sclient.NewMockClient(mockCtrl)

	scheme, err := installv1alpha1.SchemeBuilder.Build()
	if err != nil {
		t.Fatalf("should not return error when building schema")
	}

	// Binoculars
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&installv1alpha1.Binoculars{})).
		Return(nil).
		SetArg(2, expectedBinoculars)

	r := BinocularsReconciler{
		Client: mockK8sClient,
		Scheme: scheme,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{Namespace: "default", Name: "binoculars"},
	}

	_, err = r.Reconcile(context.Background(), req)
	if err == nil {
		t.Fatalf("Expected error but did not get one")
	}
}
func TestBinocularsReconciler_ReconcileGetError(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	expectedNamespacedName := types.NamespacedName{Namespace: "default", Name: "binoculars"}
	mockK8sClient := k8sclient.NewMockClient(mockCtrl)

	scheme, err := installv1alpha1.SchemeBuilder.Build()
	if err != nil {
		t.Fatalf("should not return error when building schema")
	}

	// Binoculars
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&installv1alpha1.Binoculars{})).
		Return(fmt.Errorf("Error other than notfound"))

	r := BinocularsReconciler{
		Client: mockK8sClient,
		Scheme: scheme,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{Namespace: "default", Name: "binoculars"},
	}

	_, err = r.Reconcile(context.Background(), req)
	if err == nil {
		t.Fatalf("Expected error but did not get one")
	}
}

func TestBinocularsReconciler_ReconcileUpdateError(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	expectedNamespacedName := types.NamespacedName{Namespace: "default", Name: "binoculars"}
	mockK8sClient := k8sclient.NewMockClient(mockCtrl)

	scheme, err := installv1alpha1.SchemeBuilder.Build()
	if err != nil {
		t.Fatalf("should not return error when building schema")
	}

	expectedBinoculars := v1alpha1.Binoculars{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Binoculars",
			APIVersion: "install.armadaproject.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "binoculars", DeletionTimestamp: &metav1.Time{Time: time.Now()},
			Finalizers: []string{operatorFinalizer},
		},
		Spec: v1alpha1.BinocularsSpec{
			Replicas:  2,
			HostNames: []string{"localhost"},
			Labels:    map[string]string{"test": "hello"},
			Image: installv1alpha1.Image{
				Repository: "testrepo",
				Tag:        "1.0.0",
			},
			ApplicationConfig: runtime.RawExtension{},
			ClusterIssuer:     "test",
			Ingress: &installv1alpha1.IngressConfig{
				IngressClass: "nginx",
				Labels:       map[string]string{"test": "hello"},
				Annotations:  map[string]string{"test": "hello"},
			},
		},
	}

	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&v1alpha1.Binoculars{})).
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

	// Binoculars finalizer
	mockK8sClient.EXPECT().Update(gomock.Any(), gomock.AssignableToTypeOf(&installv1alpha1.Binoculars{})).Return(fmt.Errorf("mocked update error"))

	r := BinocularsReconciler{
		Client: mockK8sClient,
		Scheme: scheme,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{Namespace: "default", Name: "binoculars"},
	}

	_, err = r.Reconcile(context.Background(), req)
	if err == nil {
		t.Fatalf("Expected error but did not get one")
	}
}
