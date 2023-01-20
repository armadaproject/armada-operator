package install

import (
	"fmt"
	"testing"
	"time"

	"context"

	install "github.com/armadaproject/armada-operator/apis/install/v1alpha1"
	"github.com/stretchr/testify/assert"

	"github.com/armadaproject/armada-operator/test/k8sclient"

	"github.com/golang/mock/gomock"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	tests := []struct {
		name    string
		jobs    []*batchv1.Job
		ctxFn   func() context.Context
		wantErr bool
	}{
		{
			name:    "it returns right away when job is complete",
			jobs:    []*batchv1.Job{sampleJobs()["complete"]},
			wantErr: false,
		},
		{
			name:    "it returns right away when job is failed",
			jobs:    []*batchv1.Job{sampleJobs()["failed"]},
			wantErr: false,
		},
		{
			name:    "it retries until the job is complete",
			jobs:    []*batchv1.Job{sampleJobs()["stuck"], sampleJobs()["stuck"], sampleJobs()["complete"]},
			wantErr: false,
		},
		{
			name:    "it retries until the job is failed",
			jobs:    []*batchv1.Job{sampleJobs()["stuck"], sampleJobs()["stuck"], sampleJobs()["failed"]},
			wantErr: false,
		},
		{
			name: "it returns an error if timeout is reached before completion",
			jobs: []*batchv1.Job{sampleJobs()["stuck"], sampleJobs()["stuck"], sampleJobs()["stuck"]},
			ctxFn: func() context.Context {
				timeoutCtx, cancelFn := context.WithTimeout(context.Background(), time.Millisecond*3)
				_ = fmt.Sprintf("ignoring cancel function to avoid timing issue: %v", cancelFn)
				return timeoutCtx
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			mockK8sClient := k8sclient.NewMockClient(mockCtrl)
			expectedNamespacedName := types.NamespacedName{Namespace: "default", Name: "lookout-migration"}
			sleepTime := time.Millisecond * 1

			for _, jb := range tt.jobs {
				mockK8sClient.
					EXPECT().
					Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&batchv1.Job{})).
					Return(nil).
					SetArg(2, *jb)
			}

			ctx := context.Background()
			if tt.ctxFn != nil {
				ctx = tt.ctxFn()
			}
			job := tt.jobs[0]
			rslt := waitForJob(ctx, mockK8sClient, job, sleepTime)
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
