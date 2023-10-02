package install

import (
	"context"
	"testing"
	"time"

	"k8s.io/utils/ptr"

	"github.com/golang/mock/gomock"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/armadaproject/armada-operator/test/k8sclient"

	"github.com/armadaproject/armada-operator/api/install/v1alpha1"
	installv1alpha1 "github.com/armadaproject/armada-operator/api/install/v1alpha1"
)

func TestArmadaServerReconciler_Reconcile(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	scheme, err := v1alpha1.SchemeBuilder.Build()
	if err != nil {
		t.Fatalf("should not return error when building schema")
	}

	expectedNS := types.NamespacedName{Namespace: "default", Name: "armadaserver"}
	waitPulsarNS := types.NamespacedName{Namespace: "default", Name: "wait-for-pulsar"}
	initPulsarNS := types.NamespacedName{Namespace: "default", Name: "init-pulsar"}

	expectedAS := installv1alpha1.ArmadaServer{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ArmadaServer",
			APIVersion: "install.armadaproject.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "armadaserver"},
		Spec: installv1alpha1.ArmadaServerSpec{
			PulsarInit: true,
			CommonSpecBase: installv1alpha1.CommonSpecBase{
				Labels: map[string]string{"test": "hello"},
				Image: installv1alpha1.Image{
					Repository: "testrepo",
					Tag:        "1.0.0",
				},
				ApplicationConfig: runtime.RawExtension{},
				Resources:         &corev1.ResourceRequirements{},
				Prometheus:        &installv1alpha1.PrometheusConfig{Enabled: true},
			},
			ClusterIssuer: "test",
			HostNames:     []string{"localhost"},
			Ingress: &installv1alpha1.IngressConfig{
				IngressClass: "nginx",
				Labels:       map[string]string{"test": "hello"},
				Annotations:  map[string]string{"test": "hello"},
			},
		},
	}

	expectedComponents, err := generateArmadaServerInstallComponents(&expectedAS, scheme)
	require.NoError(t, err)

	mockK8sClient := k8sclient.NewMockClient(mockCtrl)

	// Armada Server
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNS, gomock.AssignableToTypeOf(&installv1alpha1.ArmadaServer{})).
		Return(nil).
		SetArg(2, expectedAS)

	// Finalizer
	mockK8sClient.
		EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(&installv1alpha1.ArmadaServer{})).
		Return(nil)

	// Pulsar-wait job
	expectedComponents.Jobs[0].Status = batchv1.JobStatus{
		Conditions: []batchv1.JobCondition{{Type: batchv1.JobComplete, Status: corev1.ConditionTrue}},
	}

	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), waitPulsarNS, gomock.AssignableToTypeOf(&batchv1.Job{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "armadaserver-migration"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&batchv1.Job{})).
		Return(nil).
		SetArg(1, *expectedComponents.Jobs[0])
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), waitPulsarNS, gomock.AssignableToTypeOf(&batchv1.Job{})).
		Return(nil).
		SetArg(2, *expectedComponents.Jobs[0])

	// Pulsar-init job
	expectedComponents.Jobs[1].Status = batchv1.JobStatus{
		Conditions: []batchv1.JobCondition{{Type: batchv1.JobComplete, Status: corev1.ConditionTrue}},
	}

	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), initPulsarNS, gomock.AssignableToTypeOf(&batchv1.Job{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "armadaserver-migration"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&batchv1.Job{})).
		Return(nil).
		SetArg(1, *expectedComponents.Jobs[1])
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), initPulsarNS, gomock.AssignableToTypeOf(&batchv1.Job{})).
		Return(nil).
		SetArg(2, *expectedComponents.Jobs[1])

	// Deployment
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNS, gomock.AssignableToTypeOf(&appsv1.Deployment{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "armadaserver"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&appsv1.Deployment{})).
		Return(nil)

	ingressGRPCNamespaceNamed := types.NamespacedName{Name: "armadaserver-grpc", Namespace: "default"}
	// Ingress
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), ingressGRPCNamespaceNamed, gomock.AssignableToTypeOf(&networkingv1.Ingress{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "armadaserver"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&networkingv1.Ingress{})).
		Return(nil)

	ingressHttpNamespaceNamed := types.NamespacedName{Name: "armadaserver-rest", Namespace: "default"}
	// IngressHttp
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), ingressHttpNamespaceNamed, gomock.AssignableToTypeOf(&networkingv1.Ingress{})).
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
			CommonSpecBase: installv1alpha1.CommonSpecBase{
				Labels: nil,
				Image: installv1alpha1.Image{
					Repository: "testrepo",
					Tag:        "1.0.0",
				},
				ApplicationConfig: runtime.RawExtension{},
				Prometheus:        &installv1alpha1.PrometheusConfig{Enabled: true},
			},
			Ingress: &installv1alpha1.IngressConfig{
				IngressClass: "nginx",
			},
			HostNames: []string{"ingress.host"},
		},
	}
	mockK8sClient := k8sclient.NewMockClient(mockCtrl)
	// ArmadaServer
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&installv1alpha1.ArmadaServer{})).
		Return(nil).
		SetArg(2, expectedArmadaServer)

	// Finalizer
	mockK8sClient.
		EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(&installv1alpha1.ArmadaServer{})).
		Return(nil)

	// External cleanup
	mockK8sClient.
		EXPECT().
		Delete(gomock.Any(), gomock.AssignableToTypeOf(&monitoringv1.PrometheusRule{})).
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

func TestArmadaServerReconciler_ReconcileErrorOnApplicationConfig(t *testing.T) {
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
			CommonSpecBase: installv1alpha1.CommonSpecBase{
				Labels: nil,
				Image: installv1alpha1.Image{
					Repository: "testrepo",
					Tag:        "1.0.0",
				},
				ApplicationConfig: runtime.RawExtension{Raw: []byte(`{ "foo": "bar" `)},
				Prometheus:        &installv1alpha1.PrometheusConfig{Enabled: true},
			},
			Ingress: &installv1alpha1.IngressConfig{
				IngressClass: "nginx",
			},
		},
	}
	mockK8sClient := k8sclient.NewMockClient(mockCtrl)
	// ArmadaServer
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&installv1alpha1.ArmadaServer{})).
		Return(nil).
		SetArg(2, expectedArmadaServer)

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
	assert.Error(t, err)
}

func TestSchedulerReconciler_createIngress_EmptyHosts(t *testing.T) {
	t.Parallel()

	input := v1alpha1.ArmadaServer{}
	ingress, err := createIngressHttp(&input)
	// expect no error and nil ingress with empty hosts slice
	assert.NoError(t, err)
	assert.Nil(t, ingress)

	ingress, err = createIngressGrpc(&input)
	// expect no error and nil ingress with empty hosts slice
	assert.NoError(t, err)
	assert.Nil(t, ingress)
}

func TestSchedulerReconciler_createIngress(t *testing.T) {
	t.Parallel()

	input := v1alpha1.ArmadaServer{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Lookout",
			APIVersion: "install.armadaproject.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "lookout",
		},
		Spec: v1alpha1.ArmadaServerSpec{
			Replicas:      ptr.To[int32](2),
			ClusterIssuer: "test",
			Ingress: &v1alpha1.IngressConfig{
				IngressClass: "nginx",
			},
			HostNames: []string{"localhost"},
		},
	}
	ingress, err := createIngressHttp(&input)
	// expect no error and not-nil ingress
	assert.NoError(t, err)
	assert.NotNil(t, ingress)

	ingress, err = createIngressGrpc(&input)
	// expect no error and not-nil ingress
	assert.NoError(t, err)
	assert.NotNil(t, ingress)
}
