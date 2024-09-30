package install

import (
	"fmt"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"sigs.k8s.io/yaml"

	"context"

	"github.com/stretchr/testify/assert"

	install "github.com/armadaproject/armada-operator/api/install/v1alpha1"

	"github.com/armadaproject/armada-operator/test/k8sclient"

	"github.com/golang/mock/gomock"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

func TestImageString(t *testing.T) {
	tests := []struct {
		name     string
		Image    install.Image
		expected string
	}{
		{
			name:     "Generate Image Name",
			Image:    install.Image{Repository: "blah", Tag: "tag"},
			expected: "blah:tag",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual := ImageString(tt.Image)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestAllLabels(t *testing.T) {
	tests := []struct {
		name     string
		appName  string
		input    []map[string]string
		expected map[string]string
	}{
		{
			name:     "it merges a single labels map with boilerplate",
			appName:  "binoculars",
			input:    []map[string]string{{"hello": "world"}},
			expected: map[string]string{"hello": "world", "app": "binoculars", "release": "binoculars"},
		},
		{
			name:     "it merges multiple labels maps",
			appName:  "binoculars",
			input:    []map[string]string{{"hello": "world"}, {"hello1": "world1"}},
			expected: map[string]string{"hello": "world", "hello1": "world1", "app": "binoculars", "release": "binoculars"},
		},
		{
			name:     "it ignores nil map input",
			appName:  "binoculars",
			input:    []map[string]string{{"hello": "world"}, {"hello1": "world1"}, nil},
			expected: map[string]string{"hello": "world", "hello1": "world1", "app": "binoculars", "release": "binoculars"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual := AllLabels(tt.appName, tt.input...)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestGetConfigName(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{
			name:     "binoculars",
			expected: "binoculars-config",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual := GetConfigName(tt.name)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestIdentityLabel(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "binoculars",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual := IdentityLabel(tt.name)
			assert.Equal(t, actual["app"], tt.name)
		})
	}
}

func TestGenerateChecksumConfig(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name:     "binoculars",
			input:    []byte(`{ "test": { "foo": "bar" }}`),
			expected: "97503bec62eae4ddbc5da8c4e8743d580faf2178649fcc95be8a3f3af4ef09ca",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual := GenerateChecksumConfig(tt.input)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func Test_waitForJob(t *testing.T) {
	expectedNamespacedName := types.NamespacedName{Namespace: "default", Name: "lookout-migration"}
	tests := []struct {
		name        string
		setupMockFn func(*k8sclient.MockClient)
		ctxFn       func() context.Context
		wantErr     bool
	}{
		{
			name: "it returns right away when job is complete",
			setupMockFn: func(mockK8sClient *k8sclient.MockClient) {
				mockK8sClient.
					EXPECT().
					Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&batchv1.Job{})).
					Return(nil).
					SetArg(2, *sampleJobs()["complete"])
			},
			wantErr: false,
		},
		{
			name: "it returns right away when job is failed",
			setupMockFn: func(mockK8sClient *k8sclient.MockClient) {
				mockK8sClient.
					EXPECT().
					Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&batchv1.Job{})).
					Return(nil).
					SetArg(2, *sampleJobs()["failed"])
			},
			wantErr: false,
		},
		{
			name: "it retries until the job is complete",
			setupMockFn: func(mockK8sClient *k8sclient.MockClient) {
				jobs := []*batchv1.Job{sampleJobs()["stuck"], sampleJobs()["stuck"], sampleJobs()["complete"]}
				for _, jb := range jobs {
					mockK8sClient.
						EXPECT().
						Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&batchv1.Job{})).
						Return(nil).
						SetArg(2, *jb)
				}
			},
			wantErr: false,
		},
		{
			name: "it retries until the job is failed",
			setupMockFn: func(mockK8sClient *k8sclient.MockClient) {
				jobs := []*batchv1.Job{sampleJobs()["stuck"], sampleJobs()["stuck"], sampleJobs()["failed"]}
				for _, jb := range jobs {
					mockK8sClient.
						EXPECT().
						Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&batchv1.Job{})).
						Return(nil).
						SetArg(2, *jb)
				}
			},
			wantErr: false,
		},
		{
			name: "it returns an error if timeout is reached before completion",
			setupMockFn: func(mockK8sClient *k8sclient.MockClient) {
				job := sampleJobs()["stuck"]
				mockK8sClient.
					EXPECT().
					Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&batchv1.Job{})).
					AnyTimes().
					Return(nil).
					SetArg(2, *job)
			},
			ctxFn: func() context.Context {
				timeoutCtx, cancelFn := context.WithTimeout(context.Background(), time.Millisecond*3)
				_ = fmt.Sprintf("ignoring cancel function to avoid timing issue: %v", cancelFn)
				return timeoutCtx
			},
			wantErr: true,
		},
		{
			name: "it returns an error if get has an error",
			setupMockFn: func(mockK8sClient *k8sclient.MockClient) {
				mockK8sClient.
					EXPECT().
					Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&batchv1.Job{})).
					Return(k8serrors.NewNotFound(schema.GroupResource{}, "job"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			mockClient := k8sclient.NewMockClient(mockCtrl)
			pollInterval := time.Millisecond * 10
			timeout := time.Millisecond * 100
			tt.setupMockFn(mockClient)

			ctx := context.Background()
			if tt.ctxFn != nil {
				ctx = tt.ctxFn()
			}
			job := batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      expectedNamespacedName.Name,
					Namespace: expectedNamespacedName.Namespace,
				},
			}
			err := waitForJob(ctx, mockClient, &job, pollInterval, timeout)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}

}

func Test_isJobFinished(t *testing.T) {
	tests := []struct {
		name       string
		job        *batchv1.Job
		wantResult bool
	}{
		{
			name:       "it returns true when job has a complete status",
			job:        sampleJobs()["complete"],
			wantResult: true,
		},
		{
			name:       "it returns true when job has a failed status",
			job:        sampleJobs()["failed"],
			wantResult: true,
		},
		{
			name:       "it returns false when job lacks a terminal status",
			job:        sampleJobs()["stuck"],
			wantResult: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rslt := isJobFinished(tt.job)
			assert.Equal(t, tt.wantResult, rslt)
		})
	}
}

func Test_createEnv(t *testing.T) {
	defaultEnv := []corev1.EnvVar{
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
	}
	tests := []struct {
		name     string
		input    []corev1.EnvVar
		expected []corev1.EnvVar
	}{
		{
			name:     "with empty input expect the default",
			expected: defaultEnv,
		},
		{
			name: "with non-empty input, expect the default + the input",
			input: []corev1.EnvVar{
				{
					Name:  "ADDITIONAL",
					Value: "value",
				},
			},
			expected: append(defaultEnv, corev1.EnvVar{
				Name:  "ADDITIONAL",
				Value: "value",
			}),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual := createEnv(tt.input)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func Test_createVolumes(t *testing.T) {
	defaultVolumes := []corev1.Volume{{
		Name: volumeConfigKey,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: "secret-name",
			},
		},
	}}
	tests := []struct {
		name     string
		input    []corev1.Volume
		expected []corev1.Volume
	}{
		{
			name:     "with empty input expect the default",
			expected: defaultVolumes,
		},
		{
			name: "with non-empty input, expect the default + the input",
			input: []corev1.Volume{
				{
					Name: "ADDITIONAL",
				},
			},
			expected: append(defaultVolumes, corev1.Volume{
				Name: "ADDITIONAL",
			}),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual := createVolumes("secret-name", tt.input)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func Test_createPulsarVolumes(t *testing.T) {
	tests := []struct {
		name     string
		input    PulsarConfig
		expected []corev1.Volume
	}{
		{
			name:     "with empty pulsar config expect empty array",
			expected: nil,
		},
		{
			name: "with authentication enabled, expect token volume",
			input: PulsarConfig{
				AuthenticationEnabled: true,
			},
			expected: []corev1.Volume{{
				Name: "pulsar-token",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: "armada-pulsar-token-armada-admin",
						Items: []corev1.KeyToPath{{
							Key:  "TOKEN",
							Path: "pulsar-token",
						}},
					},
				},
			}},
		},
		{
			name: "with different secret name, use it",
			input: PulsarConfig{
				AuthenticationEnabled: true,
				AuthenticationSecret:  "some-other-secret",
			},
			expected: []corev1.Volume{{
				Name: "pulsar-token",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: "some-other-secret",
						Items: []corev1.KeyToPath{{
							Key:  "TOKEN",
							Path: "pulsar-token",
						}},
					},
				},
			}},
		},
		{
			name: "with tls enabled, expect cert volume",
			input: PulsarConfig{
				TlsEnabled: true,
			},
			expected: []corev1.Volume{{
				Name: "pulsar-ca",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: "armada-pulsar-ca-tls",
						Items: []corev1.KeyToPath{{
							Key:  "ca.crt",
							Path: "ca.crt",
						}},
					},
				},
			}},
		},
		{
			name: "with different cert, use it for secret name",
			input: PulsarConfig{
				TlsEnabled: true,
				Cacert:     "some-other-cert-name",
			},
			expected: []corev1.Volume{{
				Name: "pulsar-ca",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: "some-other-cert-name",
						Items: []corev1.KeyToPath{{
							Key:  "ca.crt",
							Path: "ca.crt",
						}},
					},
				},
			}},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual := createPulsarVolumes(tt.input)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func Test_createVolumeMount(t *testing.T) {
	defaultVolumeMounts := []corev1.VolumeMount{
		{
			Name:      volumeConfigKey,
			ReadOnly:  true,
			MountPath: "/config/application_config.yaml",
			SubPath:   "secret-name",
		},
	}
	tests := []struct {
		name     string
		input    []corev1.VolumeMount
		expected []corev1.VolumeMount
	}{
		{
			name:     "with empty input expect the default",
			expected: defaultVolumeMounts,
		},
		{
			name: "with non-empty input, expect the default + the input",
			input: []corev1.VolumeMount{
				{
					Name: "ADDITIONAL",
				},
			},
			expected: append(defaultVolumeMounts, corev1.VolumeMount{
				Name: "ADDITIONAL",
			}),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual := createVolumeMounts("secret-name", tt.input)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func Test_createPulsarVolumeMount(t *testing.T) {
	tests := []struct {
		name     string
		input    PulsarConfig
		expected []corev1.VolumeMount
	}{
		{
			name:     "with empty pulsar config expect empty array",
			expected: nil,
		},
		{
			name: "with authentication enabled, expect token volume",
			input: PulsarConfig{
				AuthenticationEnabled: true,
			},
			expected: []corev1.VolumeMount{{
				Name:      "pulsar-token",
				ReadOnly:  true,
				MountPath: "/pulsar/tokens",
			}},
		},
		{
			name: "with tls enabled, expect cert volume",
			input: PulsarConfig{
				TlsEnabled: true,
			},
			expected: []corev1.VolumeMount{{
				Name:      "pulsar-ca",
				ReadOnly:  true,
				MountPath: "/pulsar/ca",
			}},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual := createPulsarVolumeMounts(tt.input)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func sampleJobs() map[string]*batchv1.Job {
	completeJob := batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "lookout-migration",
			Namespace: "default",
		},
		Status: batchv1.JobStatus{
			Conditions: []batchv1.JobCondition{{
				Type:   batchv1.JobComplete,
				Status: corev1.ConditionTrue,
			}},
		},
	}

	failedJob := batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "lookout-migration",
			Namespace: "default",
		},
		Status: batchv1.JobStatus{
			Conditions: []batchv1.JobCondition{{
				Type:   batchv1.JobFailed,
				Status: corev1.ConditionTrue,
			}},
		},
	}

	stuckJob := batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "lookout-migration",
			Namespace: "default",
		},
		Status: batchv1.JobStatus{
			Conditions: []batchv1.JobCondition{{
				Type:   batchv1.JobComplete,
				Status: corev1.ConditionUnknown,
			}},
		},
	}

	return map[string]*batchv1.Job{"stuck": &stuckJob, "complete": &completeJob, "failed": &failedJob}
}

func TestDeepCopy(t *testing.T) {
	tests := []struct {
		name         string
		cc           CommonComponents
		expectations func(t *testing.T, old, new CommonComponents)
	}{
		{
			name: "DeepCopy clones a CommonComponents struct",
			cc:   makeCommonComponents(),
			expectations: func(t *testing.T, old, new CommonComponents) {
				assert.EqualValues(t, old.Deployment, new.Deployment)
				assert.NotSame(t, old.Deployment, new.Deployment)
				assert.Equal(t, len(old.PriorityClasses), len(new.PriorityClasses))
				assert.NotSame(t, old.PriorityClasses, new.PriorityClasses)
				assert.Equal(t, old, new)
				assert.NotSame(t, old, new)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			newCC := tt.cc.DeepCopy()
			tt.expectations(t, tt.cc, *newCC)
		})
	}
}

func TestReconcileComponents(t *testing.T) {
	initialState := makeCommonComponents()
	mutatedState := makeCommonComponents()
	newAnnotations := map[string]string{"new-annotation": "new-val"}
	mutatedState.Deployment.Annotations = newAnnotations

	tests := []struct {
		name         string
		old          CommonComponents
		new          CommonComponents
		expectations func(t *testing.T, mutated CommonComponents)
	}{
		{
			name: "DeepCopy clones a CommonComponents struct",
			old:  initialState,
			new:  mutatedState,
			expectations: func(t *testing.T, mutated CommonComponents) {
				assert.Equal(t, newAnnotations, mutated.Deployment.Annotations)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tt.old.ReconcileComponents(&tt.new)
			tt.expectations(t, tt.new)
		})
	}
}

func TestExtractPulsarConfig(t *testing.T) {

	tests := []struct {
		name     string
		input    runtime.RawExtension
		expected PulsarConfig
		wantErr  bool
	}{
		{
			name:     "it converts runtime.RawExtension json to PulsarConfig",
			input:    runtime.RawExtension{Raw: []byte(`{ "pulsar": { "tlsEnabled": true }}`)},
			expected: PulsarConfig{TlsEnabled: true},
		},
		{
			name:     "it errors if runtime.RawExtension raw is malformed json",
			input:    runtime.RawExtension{Raw: []byte(`{ "foo": "bar" `)},
			expected: PulsarConfig{},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			output, err := ExtractPulsarConfig(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, tt.expected, output)
		})
	}
}

func makeCommonComponents() CommonComponents {
	intRef := int32(5)
	deployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "some-name",
			Namespace: "some-namespace",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &intRef,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "some-name",
					Namespace: "some-namespace",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:            "armadaserver",
						ImagePullPolicy: corev1.PullIfNotPresent,
						Image:           "gresearch/someimage",
						Args:            []string{appConfigFlag, appConfigFilepath},
						Ports: []corev1.ContainerPort{{
							Name:          "metrics",
							ContainerPort: 9001,
							Protocol:      "TCP",
						}},
					}},
				},
			},
		},
	}

	automountServiceAccountToken := true
	serviceAccount := corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "some-name",
			Namespace:   "some-namespace",
			Labels:      map[string]string{"some-label-key": "some-label-value"},
			Annotations: map[string]string{"some-annotation-key": "some-annotation-value"},
		},
		ImagePullSecrets: []corev1.LocalObjectReference{
			{
				Name: "some-image-pull-secret",
			},
		},
		Secrets: []corev1.ObjectReference{
			{
				Name: "some-secret",
			},
		},
		AutomountServiceAccountToken: &automountServiceAccountToken,
	}

	pc := schedulingv1.PriorityClass{
		Value: 1000,
	}

	secret := corev1.Secret{
		StringData: map[string]string{"secretkey": "secretval"},
	}
	return CommonComponents{
		Deployment:      &deployment,
		ServiceAccount:  &serviceAccount,
		PriorityClasses: []*schedulingv1.PriorityClass{&pc},
		Secret:          &secret,
	}
}

func TestAddGoMemLimit(t *testing.T) {
	tests := []struct {
		name               string
		resourcesYaml      string
		expectedGoMemLimit string
	}{
		{
			name: "1Gi memory limit",
			resourcesYaml: `limits:
    memory: 1Gi`,
			expectedGoMemLimit: "1073741824B",
		},
		{
			name: "500Mi memory limit",
			resourcesYaml: `limits:
    memory: 500Mi`,
			expectedGoMemLimit: "524288000B",
		},
		{
			name:               "no memory limit",
			resourcesYaml:      ``,
			expectedGoMemLimit: "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resources := corev1.ResourceRequirements{}
			if err := yaml.Unmarshal([]byte(tt.resourcesYaml), &resources); err != nil {
				t.Fatalf("error unmarshalling resources yaml: %v", err)
			}

			var env []corev1.EnvVar
			env = addGoMemLimit(env, resources)

			goMemLimitFound := false
			for _, envVar := range env {
				if envVar.Name == "GOMEMLIMIT" {
					goMemLimitFound = true
					assert.Equal(t, tt.expectedGoMemLimit, envVar.Value)
				}
			}

			if !goMemLimitFound && tt.expectedGoMemLimit != "" {
				t.Errorf("expected GOMEMLIMIT to be set, but it was not found")
			}
		})
	}
}

func TestCheckAndHandleResourceDeletion(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create a logger (for tests, we use logr.Discard() to avoid actual logging)
	logger := logr.Discard()

	// Table-driven test cases
	tests := []struct {
		name              string
		deletionTimestamp *time.Time
		finalizerPresent  bool
		expectFinalizer   bool
		expectFinish      bool
		expectError       bool
	}{
		{
			name:              "Object not being deleted, finalizer not present",
			deletionTimestamp: nil,
			finalizerPresent:  false,
			expectFinalizer:   true,
			expectFinish:      false,
			expectError:       false,
		},
		{
			name:              "Object not being deleted, finalizer already present",
			deletionTimestamp: nil,
			finalizerPresent:  true,
			expectFinalizer:   true,
			expectFinish:      false,
			expectError:       false,
		},
		{
			name:              "Object being deleted, finalizer present",
			deletionTimestamp: ptr.To(time.Now()),
			finalizerPresent:  true,
			expectFinalizer:   false,
			expectFinish:      true,
			expectError:       false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a fake object
			object := newTestObject(t)

			// Set DeletionTimestamp if the test requires it
			if tc.deletionTimestamp != nil {
				object.SetDeletionTimestamp(&metav1.Time{Time: *tc.deletionTimestamp})
			}

			// Add or remove finalizer based on the test case
			finalizer := "test.finalizer"
			if tc.finalizerPresent {
				controllerutil.AddFinalizer(object, finalizer)
			}

			// Create a fake client
			fakeClient := fake.NewClientBuilder().WithObjects(object).Build()

			cleanupFunc := func(ctx context.Context) error {
				if tc.deletionTimestamp == nil {
					t.Fatalf("cleanup function should not be called")
				}
				return nil
			}

			// Call the function under test
			finish, err := checkAndHandleObjectDeletion(ctx, fakeClient, object, finalizer, cleanupFunc, logger)

			// Check for errors
			if (err != nil) != tc.expectError {
				t.Errorf("Expected error: %v, got: %v", tc.expectError, err)
			}

			// Check if reconciliation should finish
			if finish != tc.expectFinish {
				t.Errorf("Expected finish: %v, got: %v", tc.expectFinish, finish)
			}

			// Check if finalizer was added/removed as expected
			hasFinalizer := controllerutil.ContainsFinalizer(object, finalizer)
			if hasFinalizer != tc.expectFinalizer {
				t.Errorf("Expected finalizer: %v, got: %v", tc.expectFinalizer, hasFinalizer)
			}
		})
	}
}

func TestGetObjectFromCache(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// No-op logger for testing
	logger := logr.Discard()

	// Test cases
	tests := []struct {
		name         string
		objectExists bool
		expectMiss   bool
		expectError  bool
		returnError  error
	}{
		{
			name:         "Object exists in cache",
			objectExists: true,
			returnError:  nil,
			expectMiss:   false,
			expectError:  false,
		},
		{
			name:         "Object not found in cache",
			objectExists: false,
			returnError:  k8serrors.NewNotFound(newTestGroupResource(t), "test-resource"),
			expectMiss:   true,
			expectError:  false,
		},
		{
			name:         "Error while fetching from cache",
			objectExists: false,
			returnError:  errors.New("some network error"),
			expectMiss:   true,
			expectError:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a fake client with the expected error behavior
			clientBuilder := fake.NewClientBuilder()

			// Create a fake object
			object := newTestObject(t)

			if tc.objectExists {
				clientBuilder.WithObjects(object)
			}

			if tc.expectError {
				clientBuilder.WithInterceptorFuncs(interceptor.Funcs{
					Get: func(ctx context.Context, client client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
						return tc.returnError
					},
				})
			}

			fakeClient := clientBuilder.Build()

			// Call the function under test
			namespacedName := types.NamespacedName{Name: "test-resource", Namespace: "default"}
			miss, err := getObject(ctx, fakeClient, object, namespacedName, logger)
			if tc.expectError {
				assert.ErrorIs(t, err, tc.returnError)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, miss, tc.expectMiss)
		})
	}
}

// newTestObject returns a test unstructured.Unstructured object only to be used in tests.
func newTestObject(t *testing.T) *unstructured.Unstructured {
	object := &unstructured.Unstructured{}
	object.SetNamespace("default")
	object.SetName("test-resource")
	object.SetUID(uuid.NewUUID())
	object.SetGroupVersionKind(newTestGroupVersionKind(t))
	return object
}

// newTestGroupVersionKind returns a test schema.GroupVersionKind only to be used in tests.
func newTestGroupVersionKind(t *testing.T) schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   "test.group",
		Version: "v1",
		Kind:    "TestKind",
	}
}

// newTestGroupResource returns a test schema.GroupResource only to be used in tests.
func newTestGroupResource(t *testing.T) schema.GroupResource {
	return schema.GroupResource{
		Group:    "test.group",
		Resource: "test-resource",
	}
}

func TestIsNil(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    interface{}
		expected bool
	}{
		{
			name:     "Nil interface",
			input:    nil,
			expected: true,
		},
		{
			name:     "Non-nil interface with value",
			input:    42,
			expected: false,
		},
		{
			name:     "Nil pointer",
			input:    (*int)(nil),
			expected: true,
		},
		{
			name:     "Non-nil pointer",
			input:    func() *int { val := 42; return &val }(),
			expected: false,
		},
		{
			name:     "Nil slice",
			input:    ([]int)(nil),
			expected: true,
		},
		{
			name:     "Empty slice",
			input:    []int{},
			expected: false,
		},
		{
			name:     "Nil map",
			input:    (map[string]int)(nil),
			expected: true,
		},
		{
			name:     "Empty map",
			input:    map[string]int{},
			expected: false,
		},
		{
			name:     "Nil function",
			input:    (func())(nil),
			expected: true,
		},
		{
			name:     "Non-nil function",
			input:    func() {},
			expected: false,
		},
	}

	// Iterate over the test cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call the isNil function with the test input
			result := isNil(tt.input)
			if result != tt.expected {
				t.Errorf("isNil(%v) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}
