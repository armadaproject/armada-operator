package install

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/armadaproject/armada-operator/apis/install/v1alpha1"
	"github.com/armadaproject/armada-operator/test/k8sclient"

	installv1alpha1 "github.com/armadaproject/armada-operator/apis/install/v1alpha1"
)

func TestArmadaServerReconciler_Reconcile(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	expectedNS := types.NamespacedName{Namespace: "default", Name: "armadaserver"}
	expectedAS := v1alpha1.ArmadaServer{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ArmadaServer",
			APIVersion: "install.armadaproject.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "armadaserver"},
		Spec: v1alpha1.ArmadaServerSpec{
			Labels: nil,
			Image: installv1alpha1.Image{
				Repository: "testrepo",
				Tag:        "1.0.0",
			},
			ApplicationConfig: runtime.RawExtension{},
		},
	}

	mockK8sClient := k8sclient.NewMockClient(mockCtrl)

	// Armada Server
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNS, gomock.AssignableToTypeOf(&v1alpha1.ArmadaServer{})).
		Return(nil).
		SetArg(2, expectedAS)
	// Armada Server finalizer
	mockK8sClient.
		EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(&installv1alpha1.ArmadaServer{})).
		Return(nil)
	// Deployment
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNS, gomock.AssignableToTypeOf(&appsv1.Deployment{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "armadaserver"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&appsv1.Deployment{})).
		Return(nil)
	// Ingress
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNS, gomock.AssignableToTypeOf(&networkingv1.Ingress{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "armadaserver"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&networkingv1.Ingress{})).
		Return(nil)
	// IngressRest
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNS, gomock.AssignableToTypeOf(&networkingv1.Ingress{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "armadaserver"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&networkingv1.Ingress{})).
		Return(nil)
	// Service
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNS, gomock.AssignableToTypeOf(&corev1.Service{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "armadaserver"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&corev1.Service{})).
		Return(nil)
	// ServiceAccount
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNS, gomock.AssignableToTypeOf(&corev1.ServiceAccount{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "armadaserver"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&corev1.ServiceAccount{})).
		Return(nil)
	// Secret
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNS, gomock.AssignableToTypeOf(&corev1.Secret{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "armadaserver"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&corev1.Secret{})).
		Return(nil)
	// PodDisruptionBudget
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNS, gomock.AssignableToTypeOf(&policyv1.PodDisruptionBudget{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "armadaserver"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&policyv1.PodDisruptionBudget{})).
		Return(nil)
	// PrometheusRule
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNS, gomock.AssignableToTypeOf(&monitoringv1.PrometheusRule{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "armadaserver"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&monitoringv1.PrometheusRule{})).
		Return(nil)
	// ServiceMonitor
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNS, gomock.AssignableToTypeOf(&monitoringv1.ServiceMonitor{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "armadaserver"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&monitoringv1.ServiceMonitor{})).
		Return(nil)

	scheme, err := v1alpha1.SchemeBuilder.Build()
	if err != nil {
		t.Fatalf("should not return error when building schema")
	}

	r := ArmadaServerReconciler{
		Client: mockK8sClient,
		Scheme: scheme,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{Namespace: "default", Name: "armadaserver"},
	}

	_, err = r.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatalf("reconcile should not return error")
	}
}

func TestArmadaServerReconciler_ReconcileNoArmadaServer(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	expectedNamespacedName := types.NamespacedName{Namespace: "default", Name: "armadaserver-test"}
	mockK8sClient := k8sclient.NewMockClient(mockCtrl)
	// Executor
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&installv1alpha1.ArmadaServer{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "armadaserver"))
	scheme, err := installv1alpha1.SchemeBuilder.Build()
	if err != nil {
		t.Fatalf("should not return error when building schema")
	}

	r := ArmadaServerReconciler{
		Client: mockK8sClient,
		Scheme: scheme,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{Namespace: "default", Name: "armadaserver-test"},
	}

	_, err = r.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatalf("reconcile should not return error")
	}
}

func TestArmadaServerReconciler_ReconcileDeletingArmadaServer(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	expectedNamespacedName := types.NamespacedName{Namespace: "default", Name: "armadaserver"}
	expectedArmadaServer := installv1alpha1.ArmadaServer{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ArmadaServer",
			APIVersion: "install.armadaproject.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         "default",
			Name:              "armadaserver",
			DeletionTimestamp: &metav1.Time{Time: time.Now()},
			Finalizers:        []string{operatorFinalizer},
		},
		Spec: installv1alpha1.ArmadaServerSpec{
			Labels: nil,
			Image: installv1alpha1.Image{
				Repository: "testrepo",
				Tag:        "1.0.0",
			},
			ApplicationConfig: runtime.RawExtension{},
		},
	}
	mockK8sClient := k8sclient.NewMockClient(mockCtrl)
	// ArmadaServer
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&installv1alpha1.ArmadaServer{})).
		Return(nil).
		SetArg(2, expectedArmadaServer)
	// Remove ArmadaServer Finalizer
	mockK8sClient.
		EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(&installv1alpha1.ArmadaServer{})).
		Return(nil)

	scheme, err := installv1alpha1.SchemeBuilder.Build()
	if err != nil {
		t.Fatalf("should not return error when building schema")
	}

	r := ArmadaServerReconciler{
		Client: mockK8sClient,
		Scheme: scheme,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{Namespace: "default", Name: "armadaserver"},
	}

	_, err = r.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatalf("reconcile should not return error")
	}
}
