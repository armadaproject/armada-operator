package install

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"

	"github.com/armadaproject/armada-operator/internal/controller/builders"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

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

	installv1alpha1 "github.com/armadaproject/armada-operator/api/install/v1alpha1"
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
			CommonSpecBase: installv1alpha1.CommonSpecBase{
				Labels: nil,
				Image: installv1alpha1.Image{
					Repository: "testrepo",
					Tag:        "1.0.0",
				},
				ApplicationConfig: runtime.RawExtension{},
				Resources:         &corev1.ResourceRequirements{},
				Prometheus:        &installv1alpha1.PrometheusConfig{Enabled: true, ScrapeInterval: &metav1.Duration{Duration: 1 * time.Second}},
			},
		},
	}
	mockK8sClient := k8sclient.NewMockClient(mockCtrl)
	// Executor
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&installv1alpha1.Executor{})).
		Return(nil).
		SetArg(2, expectedExecutor)

	// Finalizer
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

	// PrometheusRule
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&monitoringv1.PrometheusRule{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "armadaserver"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&monitoringv1.PrometheusRule{})).
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
			CommonSpecBase: installv1alpha1.CommonSpecBase{
				Labels: nil,
				Image: installv1alpha1.Image{
					Repository: "testrepo",
					Tag:        "1.0.0",
				},
				ApplicationConfig: runtime.RawExtension{},
				Resources:         &corev1.ResourceRequirements{},
				Prometheus:        &installv1alpha1.PrometheusConfig{Enabled: true},
			},
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
	mockK8sClient.
		EXPECT().
		Delete(gomock.Any(), gomock.AssignableToTypeOf(&monitoringv1.PrometheusRule{})).
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
			CommonSpecBase: installv1alpha1.CommonSpecBase{
				Labels: nil,
				Image: installv1alpha1.Image{
					Repository: "testrepo",
					Tag:        "1.0.0",
				},
				ApplicationConfig: runtime.RawExtension{Raw: []byte(`{ "foo": "bar" `)},
				Resources:         &corev1.ResourceRequirements{},
			},
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
			CommonSpecBase: installv1alpha1.CommonSpecBase{
				CustomServiceAccount: "test-service-account",
			},
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

func TestExecutorReconciler_CreateDeployment(t *testing.T) {
	t.Parallel()

	commonConfig := &builders.CommonApplicationConfig{
		HTTPPort:    8080,
		GRPCPort:    5051,
		MetricsPort: 9000,
		Profiling: builders.ProfilingConfig{
			Port: 1337,
		},
	}

	executor := &installv1alpha1.Executor{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Executor",
			APIVersion: "install.armadaproject.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "executor"},
		Spec: installv1alpha1.ExecutorSpec{
			Replicas: ptr.To[int32](2), // Max of 1 even if configured higher
			CommonSpecBase: installv1alpha1.CommonSpecBase{
				Labels: nil,
				Image: installv1alpha1.Image{
					Repository: "testrepo",
					Tag:        "1.0.0",
				},
				ApplicationConfig: runtime.RawExtension{},
				Resources:         &corev1.ResourceRequirements{},
				Prometheus:        &installv1alpha1.PrometheusConfig{Enabled: true, ScrapeInterval: &metav1.Duration{Duration: 1 * time.Second}},
			},
		},
	}

	deployment := createExecutorDeployment(executor, "executor", commonConfig)

	expectedDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "executor",
			Namespace: "default",
			Labels: map[string]string{
				"app":     "executor",
				"release": "executor",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr.To[int32](1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "executor",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "executor",
					Namespace: "default",
					Labels: map[string]string{
						"app":     "executor",
						"release": "executor",
					},
					Annotations: map[string]string{
						"checksum/config": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
					},
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "user-config",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "executor",
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Args: []string{
								"--config",
								"/config/application_config.yaml",
							},
							Env: []corev1.EnvVar{
								{
									Name: "SERVICE_ACCOUNT",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "spec.serviceAccountName",
										},
									},
								},
								{
									Name: "POD_NAMESPACE",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.namespace",
										},
									},
								},
							},
							Image:           "testrepo:1.0.0",
							ImagePullPolicy: corev1.PullIfNotPresent,
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/health",
										Port:   intstr.FromString("http"),
										Scheme: corev1.URISchemeHTTP,
									},
								},
								InitialDelaySeconds: 10,
								TimeoutSeconds:      10,
								FailureThreshold:    3,
							},
							Name: "executor",
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: 8080,
									Protocol:      corev1.ProtocolTCP,
								},
								{
									Name:          "metrics",
									ContainerPort: 9000,
									Protocol:      corev1.ProtocolTCP,
								},
								{
									Name:          "profiling",
									ContainerPort: 1337,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/health",
										Port:   intstr.FromString("http"),
										Scheme: corev1.URISchemeHTTP,
									},
								},
								InitialDelaySeconds: 5,
								TimeoutSeconds:      5,
								FailureThreshold:    2,
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "user-config",
									ReadOnly:  true,
									MountPath: appConfigFilepath,
									SubPath:   "executor-config.yaml",
								},
							},
						},
					},
					ServiceAccountName: "executor",
				},
			},
		},
	}

	if !cmp.Equal(expectedDeployment, deployment, protocmp.Transform()) {
		t.Fatalf("deployment is not the same %s", cmp.Diff(expectedDeployment, deployment, protocmp.Transform()))
	}
}

func TestExecutorReconciler_CreateDeploymentScaledDown(t *testing.T) {
	t.Parallel()

	commonConfig := &builders.CommonApplicationConfig{
		HTTPPort:    8080,
		GRPCPort:    5051,
		MetricsPort: 9000,
		Profiling: builders.ProfilingConfig{
			Port: 1337,
		},
	}

	executor := &installv1alpha1.Executor{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Executor",
			APIVersion: "install.armadaproject.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "executor"},
		Spec: installv1alpha1.ExecutorSpec{
			Replicas: ptr.To[int32](0),
			CommonSpecBase: installv1alpha1.CommonSpecBase{
				Labels: nil,
				Image: installv1alpha1.Image{
					Repository: "testrepo",
					Tag:        "1.0.0",
				},
				ApplicationConfig: runtime.RawExtension{},
				Resources:         &corev1.ResourceRequirements{},
				Prometheus:        &installv1alpha1.PrometheusConfig{Enabled: true, ScrapeInterval: &metav1.Duration{Duration: 1 * time.Second}},
			},
		},
	}

	deployment := createExecutorDeployment(executor, "executor", commonConfig)

	assert.Equal(t, int32(0), *deployment.Spec.Replicas)
}
