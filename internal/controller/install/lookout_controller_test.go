package install

import (
	"context"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/armadaproject/armada-operator/internal/controller/builders"

	"k8s.io/utils/ptr"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/armadaproject/armada-operator/api/install/v1alpha1"
	installv1alpha1 "github.com/armadaproject/armada-operator/api/install/v1alpha1"
	"github.com/armadaproject/armada-operator/test/k8sclient"

	"github.com/golang/mock/gomock"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

func TestLookoutReconciler_ReconcilePruningDisabled(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	scheme, err := v1alpha1.SchemeBuilder.Build()
	if err != nil {
		t.Fatalf("should not return error when building schema")
	}

	expectedNamespacedName := types.NamespacedName{Namespace: "default", Name: "lookout"}
	dbPruningEnabled := false
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

	// CronJob should be deleted
	expectedCronJobName := expectedNamespacedName
	expectedCronJobName.Name = expectedCronJobName.Name + "-db-pruner"
	mockK8sClient.
		EXPECT().
		Delete(gomock.Any(), gomock.AssignableToTypeOf(&batchv1.CronJob{})).
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

func TestLookoutReconciler_CreateDeployment(t *testing.T) {
	t.Parallel()

	commonConfig := &builders.CommonApplicationConfig{
		HTTPPort:    8080,
		GRPCPort:    5051,
		MetricsPort: 9000,
		Profiling: builders.ProfilingConfig{
			Port: 1337,
		},
	}

	lookout := &v1alpha1.Lookout{
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
				ApplicationConfig: runtime.RawExtension{Raw: []byte(`{}`)},
				Resources:         &corev1.ResourceRequirements{},
				TopologyKey:       "kubernetes.io/hostname",
			},
			Replicas:      ptr.To[int32](2),
			ClusterIssuer: "test",
			Ingress: &v1alpha1.IngressConfig{
				IngressClass: "nginx",
			},
		},
	}

	deployment, err := createLookoutDeployment(lookout, "lookout", commonConfig)
	assert.NoError(t, err)

	expectedDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "lookout",
			Namespace: "default",
			Labels: map[string]string{
				"app":     "lookout",
				"release": "lookout",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr.To[int32](2),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "lookout",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "lookout",
					Namespace: "default",
					Labels: map[string]string{
						"app":     "lookout",
						"release": "lookout",
					},
					Annotations: map[string]string{
						"checksum/config": "44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a",
					},
				},
				Spec: corev1.PodSpec{
					Affinity: &corev1.Affinity{
						PodAffinity: &corev1.PodAffinity{
							PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
								{
									Weight: 100,
									PodAffinityTerm: corev1.PodAffinityTerm{
										LabelSelector: &metav1.LabelSelector{
											MatchExpressions: []metav1.LabelSelectorRequirement{
												{
													Key:      "app",
													Operator: metav1.LabelSelectorOpIn,
													Values: []string{
														"lookout",
													},
												},
											},
										},
										TopologyKey: "kubernetes.io/hostname",
									},
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "user-config",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "lookout",
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
							Name: "lookout",
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
									SubPath:   "lookout-config.yaml",
								},
							},
						},
					},
					ServiceAccountName: "lookout",
				},
			},
		},
	}

	if !cmp.Equal(expectedDeployment, deployment, protocmp.Transform()) {
		t.Fatalf("deployment is not the same %s", cmp.Diff(expectedDeployment, deployment, protocmp.Transform()))
	}
}

func TestLookoutReconciler_CreateCronJob(t *testing.T) {
	t.Parallel()

	dbPruningEnabled := true
	dbPruningSchedule := "1d"

	lookout := v1alpha1.Lookout{
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
				ApplicationConfig: runtime.RawExtension{Raw: []byte(`{}`)},
				Resources:         &corev1.ResourceRequirements{},
			},
			Replicas:      ptr.To[int32](2),
			ClusterIssuer: "test",
			Ingress: &v1alpha1.IngressConfig{
				IngressClass: "nginx",
			},
			DbPruningEnabled:  &dbPruningEnabled,
			DbPruningSchedule: &dbPruningSchedule,
		},
	}
	cronJob, err := createLookoutCronJob(&lookout, "lookout")
	assert.NoError(t, err)

	var expectedParallelism int32 = 1
	var expectedCompletions int32 = 1
	var expectedBackoffLimit int32 = 0
	var expectedTerminationGracePeriodSeconds int64 = 0

	expectedCronJob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "lookout-db-pruner",
			Namespace: "default",
			Labels: map[string]string{
				"app":     "lookout-db-pruner",
				"release": "lookout-db-pruner",
			},
			Annotations: map[string]string{
				"checksum/config": "44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a",
			},
		},
		Spec: batchv1.CronJobSpec{
			ConcurrencyPolicy: batchv1.ForbidConcurrent,
			JobTemplate: batchv1.JobTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "lookout-db-pruner",
					Namespace: "default",
					Labels: map[string]string{
						"app":     "lookout-db-pruner",
						"release": "lookout-db-pruner",
					},
				},
				Spec: batchv1.JobSpec{
					Parallelism:  &expectedParallelism,
					Completions:  &expectedCompletions,
					BackoffLimit: &expectedBackoffLimit,
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "lookout-db-pruner",
							Namespace: "default",
							Labels: map[string]string{
								"app":     "lookout-db-pruner",
								"release": "lookout-db-pruner",
							},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Args: []string{
										"--pruneDatabase",
										"--config",
										"/config/application_config.yaml",
									},
									Image:           "testrepo:1.0.0",
									ImagePullPolicy: "IfNotPresent",
									Name:            "lookout-db-pruner",
									VolumeMounts: []corev1.VolumeMount{
										{
											Name:      "user-config",
											ReadOnly:  true,
											MountPath: appConfigFilepath,
											SubPath:   "lookout-config.yaml",
										},
									},
								},
							},
							InitContainers: []corev1.Container{
								{
									Name:  "lookout-db-pruner-db-wait",
									Image: defaultAlpineImage(),
									Command: []string{
										"/bin/sh",
										"-c",
										`echo "Waiting for Postgres..."
                                                         while ! nc -z $PGHOST $PGPORT; do
                                                           sleep 1
                                                         done
                                                         echo "Postgres started!"`,
									},
									Env: []corev1.EnvVar{
										{
											Name: "PGHOST",
										},
										{
											Name: "PGPORT",
										},
									},
								},
							},
							RestartPolicy:                 "Never",
							ServiceAccountName:            "lookout",
							TerminationGracePeriodSeconds: &expectedTerminationGracePeriodSeconds,
							Volumes: []corev1.Volume{
								{
									Name: "user-config",
									VolumeSource: corev1.VolumeSource{
										Secret: &corev1.SecretVolumeSource{
											SecretName: "lookout",
										},
									},
								},
							},
						},
					},
				},
			},
			Schedule: "1d",
		},
	}

	if !cmp.Equal(expectedCronJob, cronJob, protocmp.Transform()) {
		t.Fatalf("cronjob is not the same %s", cmp.Diff(expectedCronJob, cronJob, protocmp.Transform()))
	}
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
		{
			name: "The default image is used for the db-wait init container",
			verifyJob: func(t *testing.T, job *batchv1.Job) {
				assert.Equal(t, "postgres:15.2-alpine", job.Spec.Template.Spec.InitContainers[0].Image)
			},
			wantErr: false,
		},
		{
			name: "MigrationDbWaitImage sets the image of the db-wait init container",
			modifyInput: func(cr *v1alpha1.Lookout) {
				cr.Spec.MigrationDbWaitImage = &v1alpha1.Image{
					Repository: "my-mirror.example.com/postgres",
					Tag:        "15.2-alpine",
				}
			},
			verifyJob: func(t *testing.T, job *batchv1.Job) {
				assert.Equal(t, "my-mirror.example.com/postgres:15.2-alpine", job.Spec.Template.Spec.InitContainers[0].Image)
			},
			wantErr: false,
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

func Test_createLookoutPartitionJobTableJob(t *testing.T) {
	t.Parallel()

	newLookout := func(partitionJobTable *v1alpha1.PartitionJobTableConfig) *v1alpha1.Lookout {
		return &v1alpha1.Lookout{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "lookout", Labels: map[string]string{"team": "armada"}},
			Spec: v1alpha1.LookoutSpec{
				CommonSpecBase: installv1alpha1.CommonSpecBase{
					Image:                         v1alpha1.Image{Repository: "testrepo", Tag: "1.0.0"},
					ApplicationConfig:             runtime.RawExtension{Raw: []byte(`{"postgres": {"connection": {"host": "postgres", "port": "5432"}}}`)},
					Environment:                   []corev1.EnvVar{{Name: "FOO", Value: "bar"}},
					AdditionalVolumes:             []corev1.Volume{{Name: "extra"}},
					AdditionalVolumeMounts:        []corev1.VolumeMount{{Name: "extra", MountPath: "/extra"}},
					Tolerations:                   []corev1.Toleration{{Key: "dedicated", Operator: corev1.TolerationOpExists}},
					TerminationGracePeriodSeconds: ptr.To[int64](20),
				},
				NodeSelector:         map[string]string{"pool": "lookout"},
				MigrationDbWaitImage: &v1alpha1.Image{Repository: "my-mirror.example.com/postgres", Tag: "15.2-alpine"},
				SecurityContext:      installv1alpha1.GetDefaultSecurityContext(),
				PodSecurityContext:   installv1alpha1.GetDefaultPodSecurityContext(),
				PartitionJobTable:    partitionJobTable,
			},
		}
	}

	t.Run("Job runs the partition command with the same Pod as the migration Job", func(t *testing.T) {
		t.Parallel()

		lookout := newLookout(&v1alpha1.PartitionJobTableConfig{Enabled: true})
		migrationJob, err := createLookoutMigrationJob(lookout, "sa")
		require.NoError(t, err)
		job, err := createLookoutPartitionJobTableJob(lookout, "sa")
		require.NoError(t, err)

		expectedLabels := map[string]string{"app": "lookout-partition-job-table", "release": "lookout-partition-job-table", "team": "armada"}
		assert.Equal(t, "lookout-partition-job-table", job.Name)
		assert.Equal(t, "default", job.Namespace)
		assert.Equal(t, expectedLabels, job.Labels)
		assert.Equal(t, expectedLabels, job.Spec.Template.Labels)
		assert.Equal(t, int32(0), *job.Spec.BackoffLimit)
		assert.Nil(t, job.Spec.TTLSecondsAfterFinished)
		assert.Equal(t, int64(86400), *job.Spec.ActiveDeadlineSeconds)

		podSpec := job.Spec.Template.Spec
		assert.Equal(t, corev1.RestartPolicyNever, podSpec.RestartPolicy)
		assert.Equal(t, "testrepo:1.0.0", podSpec.Containers[0].Image)
		assert.Equal(t, []string{"--partitionJobTable", "--execute", "--yes", "--config", "/config/application_config.yaml"}, podSpec.Containers[0].Args)

		expectedPodSpec := migrationJob.Spec.Template.Spec.DeepCopy()
		expectedPodSpec.InitContainers[0].Name = "lookout-partition-job-table-db-wait"
		expectedPodSpec.Containers[0].Name = "lookout-partition-job-table"
		expectedPodSpec.Containers[0].Args = podSpec.Containers[0].Args
		assert.Equal(t, *expectedPodSpec, podSpec)
	})

	t.Run("ActiveDeadlineSeconds overrides the default deadline", func(t *testing.T) {
		t.Parallel()

		job, err := createLookoutPartitionJobTableJob(newLookout(&v1alpha1.PartitionJobTableConfig{Enabled: true, ActiveDeadlineSeconds: ptr.To[int64](3600)}), "sa")
		require.NoError(t, err)
		assert.Equal(t, int64(3600), *job.Spec.ActiveDeadlineSeconds)
	})

	t.Run("bad json results in error", func(t *testing.T) {
		t.Parallel()

		lookout := newLookout(&v1alpha1.PartitionJobTableConfig{Enabled: true})
		lookout.Spec.ApplicationConfig = runtime.RawExtension{Raw: []byte(`{ "foo": "bar" `)}
		_, err := createLookoutPartitionJobTableJob(lookout, "sa")
		assert.Error(t, err)
	})
}

// recordingStatusWriter records the objects whose status is updated
type recordingStatusWriter struct {
	client.SubResourceWriter
	updated []*v1alpha1.Lookout
}

func (w *recordingStatusWriter) Update(_ context.Context, obj client.Object, _ ...client.SubResourceUpdateOption) error {
	w.updated = append(w.updated, obj.(*v1alpha1.Lookout).DeepCopy())
	return nil
}

func TestLookoutReconciler_ReconcilePartitionJobTable(t *testing.T) {
	t.Parallel()

	runningCondition := metav1.Condition{
		Type:               v1alpha1.ConditionTypeJobTablePartitioned,
		Status:             metav1.ConditionFalse,
		ObservedGeneration: 3,
		Reason:             v1alpha1.ReasonPartitionJobTableRunning,
		Message:            "Job lookout-partition-job-table is running",
	}
	newPartitionJob := func(conditions ...batchv1.JobCondition) *batchv1.Job {
		return &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "lookout-partition-job-table"},
			Status:     batchv1.JobStatus{Conditions: conditions},
		}
	}

	tests := []struct {
		name              string
		partitionJobTable *v1alpha1.PartitionJobTableConfig
		statusConditions  []metav1.Condition
		migrationJobState batchv1.JobConditionType
		// expectPartitionJob is whether the reconciler looks up the partition Job
		expectPartitionJob bool
		// existingPartitionJob is the partition Job already in the cluster, or nil if it does not exist
		existingPartitionJob *batchv1.Job
		// expectedCondition is the condition the status is updated with, or nil if the status is not updated
		expectedCondition *metav1.Condition
	}{
		{
			name:              "No partition Job when partitionJobTable is not set",
			migrationJobState: batchv1.JobComplete,
		},
		{
			name:              "No partition Job when partitionJobTable is disabled",
			partitionJobTable: &v1alpha1.PartitionJobTableConfig{Enabled: false},
			migrationJobState: batchv1.JobComplete,
		},
		{
			name:              "No partition Job when the migration Job has not succeeded, and the rest of Lookout is still installed",
			partitionJobTable: &v1alpha1.PartitionJobTableConfig{Enabled: true},
			migrationJobState: batchv1.JobFailed,
		},
		{
			name:               "Partition Job is created and reported as running once the migration Job has succeeded",
			partitionJobTable:  &v1alpha1.PartitionJobTableConfig{Enabled: true},
			migrationJobState:  batchv1.JobComplete,
			expectPartitionJob: true,
			expectedCondition:  &runningCondition,
		},
		{
			name:                 "Status is not updated when the condition is unchanged",
			partitionJobTable:    &v1alpha1.PartitionJobTableConfig{Enabled: true},
			statusConditions:     []metav1.Condition{runningCondition},
			migrationJobState:    batchv1.JobComplete,
			expectPartitionJob:   true,
			existingPartitionJob: newPartitionJob(),
		},
		{
			name:               "Succeeded partition Job is reported",
			partitionJobTable:  &v1alpha1.PartitionJobTableConfig{Enabled: true},
			statusConditions:   []metav1.Condition{runningCondition},
			migrationJobState:  batchv1.JobComplete,
			expectPartitionJob: true,
			existingPartitionJob: newPartitionJob(batchv1.JobCondition{
				Type:   batchv1.JobComplete,
				Status: corev1.ConditionTrue,
			}),
			expectedCondition: &metav1.Condition{
				Type:               v1alpha1.ConditionTypeJobTablePartitioned,
				Status:             metav1.ConditionTrue,
				ObservedGeneration: 3,
				Reason:             v1alpha1.ReasonPartitionJobTableSucceeded,
				Message:            "Job lookout-partition-job-table succeeded",
			},
		},
		{
			name:               "Failed partition Job is reported",
			partitionJobTable:  &v1alpha1.PartitionJobTableConfig{Enabled: true},
			statusConditions:   []metav1.Condition{runningCondition},
			migrationJobState:  batchv1.JobComplete,
			expectPartitionJob: true,
			existingPartitionJob: newPartitionJob(batchv1.JobCondition{
				Type:    batchv1.JobFailed,
				Status:  corev1.ConditionTrue,
				Message: "Job was active longer than specified deadline",
			}),
			expectedCondition: &metav1.Condition{
				Type:               v1alpha1.ConditionTypeJobTablePartitioned,
				Status:             metav1.ConditionFalse,
				ObservedGeneration: 3,
				Reason:             v1alpha1.ReasonPartitionJobTableFailed,
				Message:            "Job lookout-partition-job-table failed: Job was active longer than specified deadline",
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			scheme, err := v1alpha1.SchemeBuilder.Build()
			if err != nil {
				t.Fatalf("should not return error when building schema")
			}

			lookoutName := types.NamespacedName{Namespace: "default", Name: "lookout"}
			lookout := v1alpha1.Lookout{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Lookout",
					APIVersion: "install.armadaproject.io/v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace:  "default",
					Name:       "lookout",
					Generation: 3,
					Finalizers: []string{operatorFinalizer},
				},
				Spec: v1alpha1.LookoutSpec{
					CommonSpecBase: installv1alpha1.CommonSpecBase{
						Image:             v1alpha1.Image{Repository: "testrepo", Tag: "1.0.0"},
						ApplicationConfig: runtime.RawExtension{},
					},
					PartitionJobTable: tt.partitionJobTable,
				},
				Status: v1alpha1.LookoutStatus{Conditions: tt.statusConditions},
			}
			migrationJob := batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "lookout-migration"},
				Status: batchv1.JobStatus{
					Conditions: []batchv1.JobCondition{{Type: tt.migrationJobState, Status: corev1.ConditionTrue}},
				},
			}

			mockK8sClient := k8sclient.NewMockClient(mockCtrl)
			mockK8sClient.
				EXPECT().
				Get(gomock.Any(), lookoutName, gomock.AssignableToTypeOf(&v1alpha1.Lookout{})).
				Return(nil).
				SetArg(2, lookout)
			mockK8sClient.
				EXPECT().
				Get(gomock.Any(), lookoutName, gomock.AssignableToTypeOf(&corev1.ServiceAccount{})).
				Return(errors.NewNotFound(schema.GroupResource{}, "lookout"))
			mockK8sClient.
				EXPECT().
				Create(gomock.Any(), gomock.AssignableToTypeOf(&corev1.ServiceAccount{})).
				Return(nil)
			mockK8sClient.
				EXPECT().
				Get(gomock.Any(), lookoutName, gomock.AssignableToTypeOf(&corev1.Secret{})).
				Return(errors.NewNotFound(schema.GroupResource{}, "lookout"))
			mockK8sClient.
				EXPECT().
				Create(gomock.Any(), gomock.AssignableToTypeOf(&corev1.Secret{})).
				Return(nil)
			// The migration Job already exists, so it is fetched once for the upsert and once while waiting for it
			mockK8sClient.
				EXPECT().
				Get(gomock.Any(), types.NamespacedName{Namespace: "default", Name: "lookout-migration"}, gomock.AssignableToTypeOf(&batchv1.Job{})).
				Return(nil).
				SetArg(2, migrationJob).
				Times(2)
			mockK8sClient.
				EXPECT().
				Get(gomock.Any(), lookoutName, gomock.AssignableToTypeOf(&appsv1.Deployment{})).
				Return(errors.NewNotFound(schema.GroupResource{}, "lookout"))
			mockK8sClient.
				EXPECT().
				Create(gomock.Any(), gomock.AssignableToTypeOf(&appsv1.Deployment{})).
				Return(nil)
			mockK8sClient.
				EXPECT().
				Get(gomock.Any(), lookoutName, gomock.AssignableToTypeOf(&corev1.Service{})).
				Return(errors.NewNotFound(schema.GroupResource{}, "lookout"))
			mockK8sClient.
				EXPECT().
				Create(gomock.Any(), gomock.AssignableToTypeOf(&corev1.Service{})).
				Return(nil)
			mockK8sClient.
				EXPECT().
				Delete(gomock.Any(), gomock.AssignableToTypeOf(&batchv1.CronJob{})).
				Return(nil)

			var createdPartitionJob *batchv1.Job
			if tt.expectPartitionJob {
				partitionJobName := types.NamespacedName{Namespace: "default", Name: "lookout-partition-job-table"}
				if tt.existingPartitionJob != nil {
					mockK8sClient.
						EXPECT().
						Get(gomock.Any(), partitionJobName, gomock.AssignableToTypeOf(&batchv1.Job{})).
						Return(nil).
						SetArg(2, *tt.existingPartitionJob)
				} else {
					mockK8sClient.
						EXPECT().
						Get(gomock.Any(), partitionJobName, gomock.AssignableToTypeOf(&batchv1.Job{})).
						Return(errors.NewNotFound(schema.GroupResource{}, "lookout-partition-job-table"))
					mockK8sClient.
						EXPECT().
						Create(gomock.Any(), gomock.AssignableToTypeOf(&batchv1.Job{})).
						DoAndReturn(func(_ context.Context, obj client.Object, _ ...client.CreateOption) error {
							createdPartitionJob = obj.(*batchv1.Job).DeepCopy()
							return nil
						})
				}
			}

			statusWriter := &recordingStatusWriter{}
			if tt.expectedCondition != nil {
				mockK8sClient.
					EXPECT().
					Status().
					Return(statusWriter)
			}

			r := LookoutReconciler{
				Client: mockK8sClient,
				Scheme: scheme,
			}

			_, err = r.Reconcile(context.Background(), ctrl.Request{NamespacedName: lookoutName})
			assert.NoError(t, err)

			if tt.expectPartitionJob && tt.existingPartitionJob == nil {
				require.NotNil(t, createdPartitionJob)
				assert.Equal(t, "lookout-partition-job-table", createdPartitionJob.Name)
				assert.Equal(t, "lookout", createdPartitionJob.OwnerReferences[0].Name)
				assert.Equal(t, int64(86400), *createdPartitionJob.Spec.ActiveDeadlineSeconds)
			}

			if tt.expectedCondition != nil {
				require.Len(t, statusWriter.updated, 1)
				condition := meta.FindStatusCondition(statusWriter.updated[0].Status.Conditions, v1alpha1.ConditionTypeJobTablePartitioned)
				require.NotNil(t, condition)
				condition.LastTransitionTime = metav1.Time{}
				assert.Equal(t, *tt.expectedCondition, *condition)
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
