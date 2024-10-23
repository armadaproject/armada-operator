package install

import (
	"context"
	"testing"
	"time"

	"github.com/armadaproject/armada-operator/internal/controller/builders"

	"k8s.io/utils/ptr"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stretchr/testify/assert"

	"github.com/armadaproject/armada-operator/api/install/v1alpha1"
	installv1alpha1 "github.com/armadaproject/armada-operator/api/install/v1alpha1"
	"github.com/armadaproject/armada-operator/test/k8sclient"

	"github.com/golang/mock/gomock"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestSchedulerReconciler_Reconcile(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	scheme, err := v1alpha1.SchemeBuilder.Build()
	if err != nil {
		t.Fatalf("should not return error when building schema")
	}

	expectedNamespacedName := types.NamespacedName{Namespace: "default", Name: "scheduler"}
	dbPruningEnabled := true
	dbPruningSchedule := "1d"
	terminationGracePeriod := int64(20)
	expectedScheduler := v1alpha1.Scheduler{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Scheduler",
			APIVersion: "install.armadaproject.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "scheduler"},
		Spec: v1alpha1.SchedulerSpec{
			Replicas: ptr.To[int32](2),
			CommonSpecBase: installv1alpha1.CommonSpecBase{
				Labels: nil,
				Image: v1alpha1.Image{
					Repository: "testrepo",
					Tag:        "1.0.0",
				},
				ApplicationConfig:             runtime.RawExtension{},
				Resources:                     &corev1.ResourceRequirements{},
				Prometheus:                    &installv1alpha1.PrometheusConfig{Enabled: true, ScrapeInterval: &metav1.Duration{Duration: 1 * time.Second}},
				TerminationGracePeriodSeconds: &terminationGracePeriod,
			},
			ClusterIssuer: "test",
			HostNames:     []string{"localhost"},
			Ingress: &installv1alpha1.IngressConfig{
				IngressClass: "nginx",
				Labels:       map[string]string{"test": "hello"},
				Annotations:  map[string]string{"test": "hello"},
			},
			Pruner: &installv1alpha1.PrunerConfig{
				Enabled:  dbPruningEnabled,
				Schedule: dbPruningSchedule,
			},
		},
	}

	commonConfig, err := builders.ParseCommonApplicationConfig(expectedScheduler.Spec.ApplicationConfig)
	if err != nil {
		t.Fatalf("should not return error when parsing common application config")
	}
	scheduler, err := generateSchedulerInstallComponents(&expectedScheduler, scheme, commonConfig)
	if err != nil {
		t.Fatal("We should not fail on generating scheduler")
	}

	mockK8sClient := k8sclient.NewMockClient(mockCtrl)
	// Scheduler
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&v1alpha1.Scheduler{})).
		Return(nil).
		SetArg(2, expectedScheduler)

	// Finalizer
	mockK8sClient.
		EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(&installv1alpha1.Scheduler{})).
		Return(nil)

	// ServiceAccount
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&corev1.ServiceAccount{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "scheduler"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&corev1.ServiceAccount{})).
		Return(nil).
		SetArg(1, *scheduler.ServiceAccount)

	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&corev1.Secret{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "scheduler"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&corev1.Secret{})).
		Return(nil).
		SetArg(1, *scheduler.Secret)

	expectedJobName := types.NamespacedName{Namespace: "default", Name: "scheduler-migration"}
	scheduler.Jobs[0].Status = batchv1.JobStatus{
		Conditions: []batchv1.JobCondition{{
			Type:   batchv1.JobComplete,
			Status: corev1.ConditionTrue,
		}},
	}
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedJobName, gomock.AssignableToTypeOf(&batchv1.Job{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "scheduler"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&batchv1.Job{})).
		Return(nil).
		SetArg(1, *scheduler.Jobs[0])
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedJobName, gomock.AssignableToTypeOf(&batchv1.Job{})).
		Return(nil).
		SetArg(2, *scheduler.Jobs[0])

	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&appsv1.Deployment{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "scheduler"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&appsv1.Deployment{})).
		Return(nil).
		SetArg(1, *scheduler.Deployment)

	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&corev1.Service{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "scheduler"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&corev1.Service{})).
		Return(nil).
		SetArg(1, *scheduler.Service)

	// IngressGrpc
	expectedIngressName := expectedNamespacedName
	expectedIngressName.Name = expectedIngressName.Name + "-grpc"
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedIngressName, gomock.AssignableToTypeOf(&networkingv1.Ingress{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "scheduler"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&networkingv1.Ingress{})).
		Return(nil).
		SetArg(1, *scheduler.IngressGrpc)

	// ServiceMonitor
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&monitoringv1.ServiceMonitor{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "scheduler"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&monitoringv1.ServiceMonitor{})).
		Return(nil)

	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&monitoringv1.PrometheusRule{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "scheduler"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&monitoringv1.PrometheusRule{})).
		Return(nil)

	r := SchedulerReconciler{
		Client: mockK8sClient,
		Scheme: scheme,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{Namespace: "default", Name: "scheduler"},
	}

	_, err = r.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatalf("reconcile should not return error")
	}
}

func TestSchedulerReconciler_ReconcileNoScheduler(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	expectedNamespacedName := types.NamespacedName{Namespace: "default", Name: "scheduler-test"}
	mockK8sClient := k8sclient.NewMockClient(mockCtrl)
	// Executor
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&v1alpha1.Scheduler{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "scheduler"))
	scheme, err := v1alpha1.SchemeBuilder.Build()
	if err != nil {
		t.Fatalf("should not return error when building schema")
	}

	r := SchedulerReconciler{
		Client: mockK8sClient,
		Scheme: scheme,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{Namespace: "default", Name: "scheduler-test"},
	}

	_, err = r.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatalf("reconcile should not return error")
	}
}

func TestSchedulerReconciler_ReconcileErrorDueToApplicationConfig(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	expectedNamespacedName := types.NamespacedName{Namespace: "default", Name: "scheduler"}
	expectedScheduler := v1alpha1.Scheduler{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Scheduler",
			APIVersion: "install.armadaproject.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:  "default",
			Name:       "scheduler",
			Finalizers: []string{operatorFinalizer},
		},
		Spec: v1alpha1.SchedulerSpec{
			CommonSpecBase: installv1alpha1.CommonSpecBase{
				Labels: nil,
				Image: v1alpha1.Image{
					Repository: "testrepo",
					Tag:        "1.0.0",
				},
				ApplicationConfig: runtime.RawExtension{Raw: []byte(`{ "foo": "bar" `)},
				Resources:         &corev1.ResourceRequirements{},
			},
			Replicas:      ptr.To[int32](2),
			ClusterIssuer: "test",
			Ingress: &v1alpha1.IngressConfig{
				IngressClass: "nginx",
			},
		},
	}
	mockK8sClient := k8sclient.NewMockClient(mockCtrl)
	// Scheduler
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&v1alpha1.Scheduler{})).
		Return(nil).
		SetArg(2, expectedScheduler)

	scheme, err := v1alpha1.SchemeBuilder.Build()
	if err != nil {
		t.Fatalf("should not return error when building schema")
	}

	r := SchedulerReconciler{
		Client: mockK8sClient,
		Scheme: scheme,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{Namespace: "default", Name: "scheduler"},
	}

	_, err = r.Reconcile(context.Background(), req)
	assert.Error(t, err)
}

func TestSchedulerReconciler_ReconcileMissingResources(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	scheme, err := v1alpha1.SchemeBuilder.Build()
	if err != nil {
		t.Fatalf("should not return error when building schema")
	}

	expectedNamespacedName := types.NamespacedName{Namespace: "default", Name: "scheduler"}
	dbPruningEnabled := true
	dbPruningSchedule := "1d"
	terminationGracePeriod := int64(20)
	expectedScheduler := v1alpha1.Scheduler{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Scheduler",
			APIVersion: "install.armadaproject.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "scheduler"},
		Spec: v1alpha1.SchedulerSpec{
			Replicas: ptr.To[int32](2),
			CommonSpecBase: installv1alpha1.CommonSpecBase{
				Labels: nil,
				Image: v1alpha1.Image{
					Repository: "testrepo",
					Tag:        "1.0.0",
				},
				ApplicationConfig:             runtime.RawExtension{},
				Prometheus:                    &installv1alpha1.PrometheusConfig{Enabled: true, ScrapeInterval: &metav1.Duration{Duration: 1 * time.Second}},
				TerminationGracePeriodSeconds: &terminationGracePeriod,
			},
			ClusterIssuer: "test",
			HostNames:     []string{"localhost"},
			Ingress: &installv1alpha1.IngressConfig{
				IngressClass: "nginx",
				Labels:       map[string]string{"test": "hello"},
				Annotations:  map[string]string{"test": "hello"},
			},
			Pruner: &installv1alpha1.PrunerConfig{
				Enabled:  dbPruningEnabled,
				Schedule: dbPruningSchedule,
			},
		},
	}

	commonConfig, err := builders.ParseCommonApplicationConfig(expectedScheduler.Spec.ApplicationConfig)
	if err != nil {
		t.Fatalf("should not return error when parsing common application config")
	}
	scheduler, err := generateSchedulerInstallComponents(&expectedScheduler, scheme, commonConfig)
	if err != nil {
		t.Fatal("We should not fail on generating scheduler")
	}

	mockK8sClient := k8sclient.NewMockClient(mockCtrl)
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&v1alpha1.Scheduler{})).
		Return(nil).
		SetArg(2, expectedScheduler)

	// Finalizer
	mockK8sClient.
		EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(&installv1alpha1.Scheduler{})).
		Return(nil)

	// ServiceAccount
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&corev1.ServiceAccount{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "scheduler"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&corev1.ServiceAccount{})).
		Return(nil).
		SetArg(1, *scheduler.ServiceAccount)

	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&corev1.Secret{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "scheduler"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&corev1.Secret{})).
		Return(nil).
		SetArg(1, *scheduler.Secret)

	expectedJobName := types.NamespacedName{Namespace: "default", Name: "scheduler-migration"}
	scheduler.Jobs[0].Status = batchv1.JobStatus{
		Conditions: []batchv1.JobCondition{{
			Type:   batchv1.JobComplete,
			Status: corev1.ConditionTrue,
		}},
	}
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedJobName, gomock.AssignableToTypeOf(&batchv1.Job{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "scheduler"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&batchv1.Job{})).
		Return(nil).
		SetArg(1, *scheduler.Jobs[0])
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedJobName, gomock.AssignableToTypeOf(&batchv1.Job{})).
		Return(nil).
		SetArg(2, *scheduler.Jobs[0])

	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&appsv1.Deployment{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "scheduler"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&appsv1.Deployment{})).
		Return(nil).
		SetArg(1, *scheduler.Deployment)

	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&corev1.Service{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "scheduler"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&corev1.Service{})).
		Return(nil).
		SetArg(1, *scheduler.Service)

	// IngressGrpc
	expectedIngressName := expectedNamespacedName
	expectedIngressName.Name = expectedIngressName.Name + "-grpc"
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedIngressName, gomock.AssignableToTypeOf(&networkingv1.Ingress{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "scheduler"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&networkingv1.Ingress{})).
		Return(nil).
		SetArg(1, *scheduler.IngressGrpc)

	// ServiceMonitor
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&monitoringv1.ServiceMonitor{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "scheduler"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&monitoringv1.ServiceMonitor{})).
		Return(nil)

	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&monitoringv1.PrometheusRule{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "scheduler"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&monitoringv1.PrometheusRule{})).
		Return(nil)

	r := SchedulerReconciler{
		Client: mockK8sClient,
		Scheme: scheme,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{Namespace: "default", Name: "scheduler"},
	}

	_, err = r.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatalf("reconcile should not return error")
	}
}

func TestSchedulerReconciler_createSchedulerCronJob(t *testing.T) {
	t.Parallel()

	schedulerInput := v1alpha1.Scheduler{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Scheduler",
			APIVersion: "install.armadaproject.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         "default",
			Name:              "scheduler",
			DeletionTimestamp: &metav1.Time{Time: time.Now()},
			Finalizers:        []string{operatorFinalizer},
		},
		Spec: v1alpha1.SchedulerSpec{
			CommonSpecBase: installv1alpha1.CommonSpecBase{
				Labels: nil,
				Image: v1alpha1.Image{
					Repository: "testrepo",
					Tag:        "1.0.0",
				},
				ApplicationConfig: runtime.RawExtension{Raw: []byte(`{}`)},
				Resources:         &corev1.ResourceRequirements{},
			},
			Replicas:      ptr.To[int32](2),
			ClusterIssuer: "test",
			Ingress: &v1alpha1.IngressConfig{
				IngressClass: "nginx",
			},
			Pruner: &v1alpha1.PrunerConfig{
				Enabled: true,
				Args: installv1alpha1.PrunerArgs{
					Timeout:     "10m",
					Batchsize:   1000,
					ExpireAfter: "1d",
				},
				Resources: &corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						"memory": resource.Quantity{Format: "4gi"},
					},
				},
			},
		},
	}
	cronJob, err := newSchedulerCronJob(&schedulerInput, "sa")
	expectedArgs := []string{"pruneDatabase", appConfigFlag, appConfigFilepath, "--timeout", "10m", "--batchsize", "1000", "--expireAfter", "1d"}
	expectedResources := *schedulerInput.Spec.Pruner.Resources

	assert.NoError(t, err)
	assert.Equal(t, expectedArgs, cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Args)
	assert.Equal(t, expectedResources, cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Resources)
	assert.Equal(t, "sa", cronJob.Spec.JobTemplate.Spec.Template.Spec.ServiceAccountName)
}

func TestSchedulerReconciler_createSchedulerIngressGrpc_EmptyHosts(t *testing.T) {
	t.Parallel()

	input := v1alpha1.Scheduler{}
	commonConfig, err := builders.ParseCommonApplicationConfig(input.Spec.ApplicationConfig)
	if err != nil {
		t.Fatalf("should not return error when parsing common application config")
	}
	ingress, err := newSchedulerIngressGRPC(&input, commonConfig)
	// expect no error and nil ingress with empty hosts slice
	assert.NoError(t, err)
	assert.Nil(t, ingress)
}

func TestSchedulerReconciler_createSchedulerIngressGrpc(t *testing.T) {
	t.Parallel()

	input := v1alpha1.Scheduler{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Scheduler",
			APIVersion: "install.armadaproject.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "scheduler",
		},
		Spec: v1alpha1.SchedulerSpec{
			Replicas:      ptr.To[int32](2),
			ClusterIssuer: "test",
			Ingress: &v1alpha1.IngressConfig{
				IngressClass: "nginx",
			},
			HostNames: []string{"localhost"},
		},
	}
	commonConfig, err := builders.ParseCommonApplicationConfig(input.Spec.ApplicationConfig)
	if err != nil {
		t.Fatalf("should not return error when parsing common application config")
	}
	ingress, err := newSchedulerIngressGRPC(&input, commonConfig)
	// expect no error and not-nil ingress
	assert.NoError(t, err)
	assert.NotNil(t, ingress)
}

func TestSchedulerReconciler_createSchedulerCronJobError(t *testing.T) {
	t.Parallel()

	expectedScheduler := v1alpha1.Scheduler{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Scheduler",
			APIVersion: "install.armadaproject.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         "default",
			Name:              "scheduler",
			DeletionTimestamp: &metav1.Time{Time: time.Now()},
			Finalizers:        []string{operatorFinalizer},
		},
		Spec: v1alpha1.SchedulerSpec{
			CommonSpecBase: installv1alpha1.CommonSpecBase{
				Labels: nil,
				Image: v1alpha1.Image{
					Repository: "testrepo",
					Tag:        "1.0.0",
				},
				ApplicationConfig: runtime.RawExtension{Raw: []byte(`{ "foo": "bar" `)},
				Resources:         &corev1.ResourceRequirements{},
			},
			Replicas:      ptr.To[int32](2),
			ClusterIssuer: "test",
			Ingress: &v1alpha1.IngressConfig{
				IngressClass: "nginx",
			},
		},
	}
	_, err := newSchedulerCronJob(&expectedScheduler, "scheduler")
	assert.Error(t, err)
	assert.Equal(t, "yaml: line 1: did not find expected ',' or '}'", err.Error())
}

func TestSchedulerReconciler_ReconcileDeletingScheduler(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	expectedNamespacedName := types.NamespacedName{Namespace: "default", Name: "scheduler"}
	dbPruningEnabled := true

	expectedScheduler := v1alpha1.Scheduler{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Scheduler",
			APIVersion: "install.armadaproject.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         "default",
			Name:              "scheduler",
			DeletionTimestamp: &metav1.Time{Time: time.Now()},
			Finalizers:        []string{operatorFinalizer},
		},
		Spec: v1alpha1.SchedulerSpec{
			HostNames: []string{"ingress.host"},
			CommonSpecBase: installv1alpha1.CommonSpecBase{
				Labels: nil,
				Image: v1alpha1.Image{
					Repository: "testrepo",
					Tag:        "1.0.0",
				},
				ApplicationConfig: runtime.RawExtension{},
				Resources:         &corev1.ResourceRequirements{},
				Prometheus:        &installv1alpha1.PrometheusConfig{Enabled: true},
			},
			Replicas:      ptr.To[int32](2),
			ClusterIssuer: "test",
			Ingress: &v1alpha1.IngressConfig{
				IngressClass: "nginx",
			},
			Pruner: &installv1alpha1.PrunerConfig{
				Enabled: dbPruningEnabled,
			},
		},
	}
	mockK8sClient := k8sclient.NewMockClient(mockCtrl)
	// Scheduler
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&v1alpha1.Scheduler{})).
		Return(nil).
		SetArg(2, expectedScheduler)

	// Finalizer
	mockK8sClient.
		EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(&installv1alpha1.Scheduler{})).
		Return(nil)

	scheme, err := v1alpha1.SchemeBuilder.Build()
	if err != nil {
		t.Fatalf("should not return error when building schema")
	}

	r := SchedulerReconciler{
		Client: mockK8sClient,
		Scheme: scheme,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{Namespace: "default", Name: "scheduler"},
	}

	_, err = r.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatalf("reconcile should not return error")
	}
}

func TestSchedulerReconciler_ReconcileDeletingSchedulerWithError(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	expectedNamespacedName := types.NamespacedName{Namespace: "default", Name: "scheduler"}

	mockK8sClient := k8sclient.NewMockClient(mockCtrl)
	// Scheduler
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&v1alpha1.Scheduler{})).
		Return(errors.NewBadRequest("something is amiss"))

	scheme, err := v1alpha1.SchemeBuilder.Build()
	if err != nil {
		t.Fatalf("should not return error when building schema")
	}

	r := SchedulerReconciler{
		Client: mockK8sClient,
		Scheme: scheme,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{Namespace: "default", Name: "scheduler"},
	}

	_, err = r.Reconcile(context.Background(), req)
	if err == nil {
		t.Fatalf("reconcile should return error")
	}
}

func Test_createSchedulerMigrationJob(t *testing.T) {
	tests := []struct {
		name        string
		modifyInput func(*v1alpha1.Scheduler)
		verifyJob   func(*testing.T, *batchv1.Job)
		wantErr     bool
	}{
		{
			name: "Postgres properties are extracted from AppConfig",
			modifyInput: func(cr *v1alpha1.Scheduler) {
				json := `{"postgres": {"connection": {"host": "postgres3000", "port": "4000"}}}`
				bytes := []byte(json)
				cr.Spec.ApplicationConfig = runtime.RawExtension{
					Raw: bytes,
				}
			},
			verifyJob: func(t *testing.T, job *batchv1.Job) {
				assert.Equal(t, "PGHOST", job.Spec.Template.Spec.InitContainers[0].Env[0].Name)
				assert.Equal(t, "postgres3000", job.Spec.Template.Spec.InitContainers[0].Env[0].Value)
				assert.Equal(t, "PGPORT", job.Spec.Template.Spec.InitContainers[0].Env[1].Name)
				assert.Equal(t, "4000", job.Spec.Template.Spec.InitContainers[0].Env[1].Value)
				assert.Equal(t, "sa", job.Spec.Template.Spec.ServiceAccountName)
			},
			wantErr: false,
		},
		{
			name: "Missing app config properties result in empty string values, not error",
			modifyInput: func(cr *v1alpha1.Scheduler) {
				json := `{"postgres": {"connection": {"hostWrongKey": "postgres3000", "portWrongKey": "4000"}}}`
				bytes := []byte(json)
				cr.Spec.ApplicationConfig = runtime.RawExtension{
					Raw: bytes,
				}
			},
			verifyJob: func(t *testing.T, job *batchv1.Job) {
				assert.Equal(t, "PGHOST", job.Spec.Template.Spec.InitContainers[0].Env[0].Name)
				assert.Equal(t, "", job.Spec.Template.Spec.InitContainers[0].Env[0].Value)
				assert.Equal(t, "PGPORT", job.Spec.Template.Spec.InitContainers[0].Env[1].Name)
				assert.Equal(t, "", job.Spec.Template.Spec.InitContainers[0].Env[1].Value)
				assert.Equal(t, "sa", job.Spec.Template.Spec.ServiceAccountName)
			},
			wantErr: false,
		},
		{
			name: "bad json results in error",
			modifyInput: func(cr *v1alpha1.Scheduler) {
				json := `{"postgres": {"connection": ["hostWrongKey": "postgres3000", "portWrongKey": "4000"}}}`
				bytes := []byte(json)
				cr.Spec.ApplicationConfig = runtime.RawExtension{
					Raw: bytes,
				}
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cr := v1alpha1.Scheduler{}
			if tt.modifyInput != nil {
				tt.modifyInput(&cr)
			}
			rslt, err := newSchedulerMigrationJob(&cr, "sa")

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.verifyJob != nil {
				tt.verifyJob(t, rslt)
			}
		})
	}
}
