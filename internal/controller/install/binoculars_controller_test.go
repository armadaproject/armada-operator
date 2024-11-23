package install

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/armadaproject/armada-operator/internal/controller/builders"

	"k8s.io/utils/ptr"

	"github.com/stretchr/testify/assert"

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

	"github.com/armadaproject/armada-operator/api/install/v1alpha1"
	installv1alpha1 "github.com/armadaproject/armada-operator/api/install/v1alpha1"
)

func TestBinoculars_GenerateBinocularsInstallComponents(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	scheme, err := installv1alpha1.SchemeBuilder.Build()
	if err != nil {
		t.Fatalf("should not return error when building schema")
	}
	tests := map[string]struct {
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
					Replicas:      ptr.To[int32](2),
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
					Replicas:      ptr.To[int32](2),
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

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			commonConfig, err := builders.ParseCommonApplicationConfig(tt.binoculars.Spec.ApplicationConfig)
			if tt.expectedError {
				assert.Error(t, err)
				return
			} else {
				assert.NoError(t, err)
			}
			_, err = generateBinocularsInstallComponents(tt.binoculars, scheme, commonConfig)
			assert.NoError(t, err)
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

			Replicas:      ptr.To[int32](2),
			HostNames:     []string{"localhost"},
			ClusterIssuer: "test",
			Ingress: &installv1alpha1.IngressConfig{
				IngressClass: "nginx",
				Labels:       map[string]string{"test": "hello"},
				Annotations:  map[string]string{"test": "hello"},
			},
		},
	}

	commonConfig, err := builders.ParseCommonApplicationConfig(expectedBinoculars.Spec.ApplicationConfig)
	if err != nil {
		t.Fatalf("should not return error when parsing common application config")
	}
	binoculars, err := generateBinocularsInstallComponents(&expectedBinoculars, scheme, commonConfig)
	if err != nil {
		t.Fatal("We should not fail on generating binoculars")
	}

	// Binoculars
	mockK8sClient := k8sclient.NewMockClient(mockCtrl)
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&v1alpha1.Binoculars{})).
		Return(nil).
		SetArg(2, expectedBinoculars)

	// Finalizer
	mockK8sClient.
		EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(&installv1alpha1.Binoculars{})).
		Return(nil)

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
				Resources:         &corev1.ResourceRequirements{},
			},
			Replicas:      ptr.To[int32](2),
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
				Resources:         &corev1.ResourceRequirements{},
			},
			Replicas:      ptr.To[int32](2),
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

func TestSchedulerReconciler_createBinolcularsIngress_EmptyHosts(t *testing.T) {
	t.Parallel()

	input := v1alpha1.Binoculars{}
	commonConfig, err := builders.ParseCommonApplicationConfig(input.Spec.ApplicationConfig)
	if err != nil {
		t.Fatalf("should not return error when parsing common application config")
	}
	ingress, err := createBinocularsIngressHttp(&input, commonConfig)
	// expect no error and nil ingress with empty hosts slice
	assert.NoError(t, err)
	assert.Nil(t, ingress)

	ingress, err = createBinocularsIngressGrpc(&input, commonConfig)
	// expect no error and nil ingress with empty hosts slice
	assert.NoError(t, err)
	assert.Nil(t, ingress)
}

func TestSchedulerReconciler_createBinocularsIngress(t *testing.T) {
	t.Parallel()

	input := v1alpha1.Binoculars{
		TypeMeta: metav1.TypeMeta{
			Kind:       "binoculars",
			APIVersion: "install.armadaproject.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "binoculars",
		},
		Spec: v1alpha1.BinocularsSpec{
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
	ingress, err := createBinocularsIngressHttp(&input, commonConfig)
	// expect no error and not-nil ingress
	assert.NoError(t, err)
	assert.NotNil(t, ingress)

	ingress, err = createBinocularsIngressGrpc(&input, commonConfig)
	// expect no error and not-nil ingress
	assert.NoError(t, err)
	assert.NotNil(t, ingress)
}

func TestBinocularsReconciler_CreateDeployment(t *testing.T) {
	t.Parallel()

	commonConfig := &builders.CommonApplicationConfig{
		HTTPPort:    8080,
		GRPCPort:    5051,
		MetricsPort: 9000,
		Profiling: builders.ProfilingConfig{
			Port: 1337,
		},
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "binoculars",
		},
	}

	binoculars := &v1alpha1.Binoculars{
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

			Replicas:      ptr.To[int32](2),
			HostNames:     []string{"localhost"},
			ClusterIssuer: "test",
			Ingress: &installv1alpha1.IngressConfig{
				IngressClass: "nginx",
				Labels:       map[string]string{"test": "hello"},
				Annotations:  map[string]string{"test": "hello"},
			},
		},
	}

	deployment, err := createBinocularsDeployment(binoculars, secret, "binoculars", commonConfig)
	assert.NoError(t, err)

	expectedDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "binoculars",
			Namespace: "default",
			Labels: map[string]string{
				"app":     "binoculars",
				"release": "binoculars",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr.To[int32](2),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "binoculars",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "binoculars",
					Namespace: "default",
					Labels: map[string]string{
						"app":     "binoculars",
						"release": "binoculars",
					},
					Annotations: map[string]string{
						"checksum/config": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
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
														"binoculars",
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
									SecretName: "binoculars",
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
							Name: "binoculars",
							Ports: []corev1.ContainerPort{
								{
									Name:          "grpc",
									ContainerPort: 5051,
									Protocol:      corev1.ProtocolTCP,
								},
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
									SubPath:   "binoculars-config.yaml",
								},
							},
						},
					},
					ServiceAccountName: "binoculars",
				},
			},
		},
	}

	if !cmp.Equal(expectedDeployment, deployment, protocmp.Transform()) {
		t.Fatalf("deployment is not the same %s", cmp.Diff(expectedDeployment, deployment, protocmp.Transform()))
	}
}
