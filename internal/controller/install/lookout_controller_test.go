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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestLookoutReconciler_Reconcile(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	scheme, err := v1alpha1.SchemeBuilder.Build()
	if err != nil {
		t.Fatalf("should not return error when building schema")
	}

	expectedNamespacedName := types.NamespacedName{Namespace: "default", Name: "lookout"}
	dbPruningEnabled := true
	dbPruningSchedule := "1d"
	terminationGracePeriod := int64(20)
	expectedLookout := v1alpha1.Lookout{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Lookout",
			APIVersion: "install.armadaproject.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "lookout"},
		Spec: v1alpha1.LookoutSpec{
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
			DbPruningEnabled:  &dbPruningEnabled,
			DbPruningSchedule: &dbPruningSchedule,
		},
	}

	mockK8sClient := k8sclient.NewMockClient(mockCtrl)
	// Lookout
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&v1alpha1.Lookout{})).
		Return(nil).
		SetArg(2, expectedLookout)

	// Finalizer
	mockK8sClient.
		EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(&installv1alpha1.Lookout{})).
		Return(nil)

	// ServiceAccount
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&corev1.ServiceAccount{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "lookout"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&corev1.ServiceAccount{})).
		Return(nil)

	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&corev1.Secret{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "lookout"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&corev1.Secret{})).
		Return(nil)

	expectedJobName := types.NamespacedName{Namespace: "default", Name: "lookout-migration"}
	expectedMigrationJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "lookout-migration",
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "lookout-migration",
							Image: "testrepo:1.0.0",
						},
					},
				},
			},
		},
		Status: batchv1.JobStatus{
			Conditions: []batchv1.JobCondition{{
				Type:   batchv1.JobComplete,
				Status: corev1.ConditionTrue,
			}},
		},
	}

	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedJobName, gomock.AssignableToTypeOf(&batchv1.Job{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "lookout"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&batchv1.Job{})).
		Return(nil)
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedJobName, gomock.AssignableToTypeOf(&batchv1.Job{})).
		Return(nil).
		SetArg(2, *expectedMigrationJob)

	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&appsv1.Deployment{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "lookout"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&appsv1.Deployment{})).
		Return(nil)

	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&corev1.Service{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "lookout"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&corev1.Service{})).
		Return(nil)

	// IngressHttp
	expectedIngressName := expectedNamespacedName
	expectedIngressName.Name = expectedIngressName.Name + "-rest"
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedIngressName, gomock.AssignableToTypeOf(&networkingv1.Ingress{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "lookout"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&networkingv1.Ingress{})).
		Return(nil)

	// ServiceMonitor
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&monitoringv1.ServiceMonitor{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "armadaserver"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&monitoringv1.ServiceMonitor{})).
		Return(nil)

	// CronJob
	expectedCronJobName := expectedNamespacedName
	expectedCronJobName.Name = expectedCronJobName.Name + "-db-pruner"
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedCronJobName, gomock.AssignableToTypeOf(&batchv1.CronJob{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "armadaserver"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&batchv1.CronJob{})).
		Return(nil)

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
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&v1alpha1.Lookout{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "lookout"))
	scheme, err := v1alpha1.SchemeBuilder.Build()
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

func TestLookoutReconciler_ReconcileErrorDueToApplicationConfig(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	expectedNamespacedName := types.NamespacedName{Namespace: "default", Name: "lookout"}
	expectedLookout := v1alpha1.Lookout{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Lookout",
			APIVersion: "install.armadaproject.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:  "default",
			Name:       "lookout",
			Finalizers: []string{operatorFinalizer},
		},
		Spec: v1alpha1.LookoutSpec{
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
	// Lookout
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&v1alpha1.Lookout{})).
		Return(nil).
		SetArg(2, expectedLookout)

	scheme, err := v1alpha1.SchemeBuilder.Build()
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
	assert.Error(t, err)
}

func TestLookoutReconciler_CreateCronJobErrorDueToApplicationConfig(t *testing.T) {
	t.Parallel()

	expectedLookout := v1alpha1.Lookout{
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
		Spec: v1alpha1.LookoutSpec{
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
	_, err := createLookoutCronJob(&expectedLookout, "lookout")
	assert.Error(t, err)
	assert.Equal(t, "yaml: line 1: did not find expected ',' or '}'", err.Error())
}

func TestLookoutReconciler_ReconcileDeletingLookout(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	expectedNamespacedName := types.NamespacedName{Namespace: "default", Name: "lookout"}
	dbPruningEnabled := true

	expectedLookout := v1alpha1.Lookout{
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
		Spec: v1alpha1.LookoutSpec{
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
			DbPruningEnabled: &dbPruningEnabled,
		},
	}
	mockK8sClient := k8sclient.NewMockClient(mockCtrl)
	// Lookout
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&v1alpha1.Lookout{})).
		Return(nil).
		SetArg(2, expectedLookout)

	// Finalizer
	mockK8sClient.
		EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(&installv1alpha1.Lookout{})).
		Return(nil)

	scheme, err := v1alpha1.SchemeBuilder.Build()
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

func TestLookoutReconciler_ReconcileDeletingLookoutWithError(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	expectedNamespacedName := types.NamespacedName{Namespace: "default", Name: "lookout"}
	dbPruningEnabled := true

	expectedLookout := v1alpha1.Lookout{
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
		Spec: v1alpha1.LookoutSpec{
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
			DbPruningEnabled: &dbPruningEnabled,
		},
	}
	mockK8sClient := k8sclient.NewMockClient(mockCtrl)
	// Lookout
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&v1alpha1.Lookout{})).
		Return(nil).
		SetArg(2, expectedLookout)

	// Finalizer update will error
	mockK8sClient.
		EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(&installv1alpha1.Lookout{})).
		Return(errors.NewResourceExpired("this finalizer does not exist"))

	scheme, err := v1alpha1.SchemeBuilder.Build()
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
	if err == nil {
		t.Fatalf("reconcile should return error")
	}
}

func Test_createLookoutMigrationJob(t *testing.T) {
	tests := []struct {
		name        string
		modifyInput func(*v1alpha1.Lookout)
		verifyJob   func(*testing.T, *batchv1.Job)
		wantErr     bool
	}{
		{
			name: "Postgres properties are extracted from AppConfig",
			modifyInput: func(cr *v1alpha1.Lookout) {
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
			modifyInput: func(cr *v1alpha1.Lookout) {
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
			modifyInput: func(cr *v1alpha1.Lookout) {
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

			cr := v1alpha1.Lookout{}
			if tt.modifyInput != nil {
				tt.modifyInput(&cr)
			}
			rslt, err := createLookoutMigrationJob(&cr, "sa")

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

func TestSchedulerReconciler_createIngressHttp_EmptyHosts(t *testing.T) {
	t.Parallel()

	input := v1alpha1.Lookout{}
	commonConfig, err := builders.ParseCommonApplicationConfig(input.Spec.ApplicationConfig)
	if err != nil {
		t.Fatalf("should not return error when parsing common application config")
	}
	ingress, err := createLookoutIngressHttp(&input, commonConfig)
	// expect no error and nil ingress with empty hosts slice
	assert.NoError(t, err)
	assert.Nil(t, ingress)
}

func TestSchedulerReconciler_createLookoutIngressHttp(t *testing.T) {
	t.Parallel()

	input := v1alpha1.Lookout{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Lookout",
			APIVersion: "install.armadaproject.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "lookout",
		},
		Spec: v1alpha1.LookoutSpec{
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
	ingress, err := createLookoutIngressHttp(&input, commonConfig)
	// expect no error and not-nil ingress
	assert.NoError(t, err)
	assert.NotNil(t, ingress)
}
