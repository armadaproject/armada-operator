package install

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/armadaproject/armada-operator/apis/install/v1alpha1"
	installv1alpha1 "github.com/armadaproject/armada-operator/apis/install/v1alpha1"
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
	expectedLookout := v1alpha1.Lookout{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Lookout",
			APIVersion: "install.armadaproject.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "lookout"},
		Spec: v1alpha1.LookoutSpec{
			Replicas: 2,
			Labels:   nil,
			Image: v1alpha1.Image{
				Repository: "testrepo",
				Tag:        "1.0.0",
			},
			ApplicationConfig: runtime.RawExtension{},
			ClusterIssuer:     "test",
			HostNames:         []string{"localhost"},
			Ingress: &installv1alpha1.IngressConfig{
				IngressClass: "nginx",
				Labels:       map[string]string{"test": "hello"},
				Annotations:  map[string]string{"test": "hello"},
			},
			Resources: &corev1.ResourceRequirements{},
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

	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&corev1.Secret{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "lookout"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&corev1.Secret{})).
		Return(nil).
		SetArg(1, *lookout.Secret)

	expectedJobName := types.NamespacedName{Namespace: "default", Name: "lookout-migration"}
	lookout.Job.Status = batchv1.JobStatus{
		Conditions: []batchv1.JobCondition{{
			Type:   batchv1.JobComplete,
			Status: corev1.ConditionTrue,
		}},
	}
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedJobName, gomock.AssignableToTypeOf(&batchv1.Job{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "lookout"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&batchv1.Job{})).
		Return(nil).
		SetArg(1, *lookout.Job)
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedJobName, gomock.AssignableToTypeOf(&batchv1.Job{})).
		Return(nil).
		SetArg(2, *lookout.Job)

	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&appsv1.Deployment{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "lookout"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&appsv1.Deployment{})).
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

	// IngressWeb
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&networkingv1.Ingress{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "lookout"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&networkingv1.Ingress{})).
		Return(nil).
		SetArg(1, *lookout.IngressWeb)

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
			Namespace:         "default",
			Name:              "lookout",
			DeletionTimestamp: &metav1.Time{Time: time.Now()},
			Finalizers:        []string{operatorFinalizer},
		},
		Spec: v1alpha1.LookoutSpec{
			Replicas: 2,
			Labels:   nil,
			Image: v1alpha1.Image{
				Repository: "testrepo",
				Tag:        "1.0.0",
			},
			ApplicationConfig: runtime.RawExtension{Raw: []byte(`{ "foo": "bar" `)},
			ClusterIssuer:     "test",
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

func TestLookoutReconciler_ReconcileDeletingLookout(t *testing.T) {
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
			Namespace:         "default",
			Name:              "lookout",
			DeletionTimestamp: &metav1.Time{Time: time.Now()},
			Finalizers:        []string{operatorFinalizer},
		},
		Spec: v1alpha1.LookoutSpec{
			Replicas: 2,
			Labels:   nil,
			Image: v1alpha1.Image{
				Repository: "testrepo",
				Tag:        "1.0.0",
			},
			ApplicationConfig: runtime.RawExtension{},
			ClusterIssuer:     "test",
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
	if err != nil {
		t.Fatalf("reconcile should not return error")
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
		t.Run(tt.name, func(t *testing.T) {
			cr := v1alpha1.Lookout{}
			if tt.modifyInput != nil {
				tt.modifyInput(&cr)
			}
			rslt, err := createLookoutMigrationJob(&cr)

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
