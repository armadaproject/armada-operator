package install

import (
	"fmt"
	"testing"
	"time"

	"context"

	"github.com/stretchr/testify/assert"

	install "github.com/armadaproject/armada-operator/apis/install/v1alpha1"

	"github.com/armadaproject/armada-operator/test/k8sclient"

	"github.com/golang/mock/gomock"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		t.Run(tt.name, func(t *testing.T) {
			actual := ImageString(tt.Image)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestAllLabels(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]string
		expected map[string]string
	}{
		{
			name:     "binoculars",
			input:    map[string]string{"hello": "world"},
			expected: map[string]string{"hello": "world", "app": "binoculars", "release": "binoculars"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := AllLabels(tt.name, tt.input)
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
		t.Run(tt.name, func(t *testing.T) {
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
		t.Run(tt.name, func(t *testing.T) {
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
		t.Run(tt.name, func(t *testing.T) {
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
					Return(errors.NewNotFound(schema.GroupResource{}, "job"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			mockK8sClient := k8sclient.NewMockClient(mockCtrl)
			sleepTime := time.Millisecond * 1
			tt.setupMockFn(mockK8sClient)

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
			rslt := waitForJob(ctx, mockK8sClient, &job, sleepTime)
			if tt.wantErr {
				assert.Error(t, rslt)
			} else {
				assert.NoError(t, rslt)
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
		t.Run(tt.name, func(t *testing.T) {
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
		t.Run(tt.name, func(t *testing.T) {
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
		t.Run(tt.name, func(t *testing.T) {
			actual := createVolumes("secret-name", tt.input)
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
		t.Run(tt.name, func(t *testing.T) {
			actual := createVolumeMounts("secret-name", tt.input)
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
		t.Run(tt.name, func(t *testing.T) {
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
		t.Run(tt.name, func(t *testing.T) {
			tt.old.ReconcileComponents(&tt.new)
			tt.expectations(t, tt.new)
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
						ImagePullPolicy: "IfNotPresent",
						Image:           "gresearch/someimage",
						Args:            []string{"--config", "/config/application_config.yaml"},
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

	pc := schedulingv1.PriorityClass{
		Value: 1000,
	}

	secret := corev1.Secret{
		StringData: map[string]string{"secretkey": "secretval"},
	}

	return CommonComponents{
		Deployment:      &deployment,
		PriorityClasses: []*schedulingv1.PriorityClass{&pc},
		Secret:          &secret,
	}
}
