package webhook

import (
	"context"
	"testing"

	v1alpha "github.com/armadaproject/armada-operator/apis/install/v1alpha1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestExecutorDefaultWebhookTest(t *testing.T) {
	tests := []struct {
		name           string
		repository     string
		resources      *corev1.ResourceRequirements
		scrapeInterval string
	}{
		{
			name:           "correctly initialize defaults",
			repository:     "",
			scrapeInterval: "",
			resources:      nil,
		},
		{
			name:       "specify repository",
			repository: "blah",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			webhook := ExecutorWebhook{}
			executor := executorSpec(tc.repository, tc.resources, tc.scrapeInterval)
			err := webhook.Default(context.Background(), executor)
			assert.NoError(t, err)
			if tc.repository != "" {
				assert.Equal(t, executor.Spec.Image.Repository, tc.repository)
			} else {
				assert.Equal(t, executor.Spec.Image.Repository, "gresearchdev/armada-executor")
			}
			if tc.resources != nil {
				assert.Equal(t, executor.Spec.Resources, tc.resources)
			} else {
				assert.Equal(t, executor.Spec.Resources, &corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						"cpu":    resource.MustParse("300m"),
						"memory": resource.MustParse("1Gi"),
					},
					Requests: corev1.ResourceList{
						"cpu":    resource.MustParse("200m"),
						"memory": resource.MustParse("512Mi"),
					},
				})
			}
			if tc.scrapeInterval != "" {
				assert.Equal(t, executor.Spec.Prometheus.ScrapeInterval, tc.scrapeInterval)
			} else {
				assert.Equal(t, executor.Spec.Prometheus.ScrapeInterval, "10s")
			}
		})
	}
}

func executorSpec(repository string, resources *corev1.ResourceRequirements, scrapeInterval string) *v1alpha.Executor {
	executor := &v1alpha.Executor{
		Spec: v1alpha.ExecutorSpec{},
	}
	if repository != "" {
		executor.Spec.Image.Repository = repository
	}
	if resources != nil {
		executor.Spec.Resources = resources
	}
	if scrapeInterval != "" {
		executor.Spec.Prometheus.ScrapeInterval = scrapeInterval
	}
	return executor
}

func TestExecutorValidateDelete(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "server validate delete",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			executorWebhook := ExecutorWebhook{}
			executor := &v1alpha.Executor{}
			assert.NoError(t, executorWebhook.ValidateDelete(context.Background(), executor))
		})
	}
}

func TestExecutorValidateUpdate(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "executor validate update",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			executorWebhook := ExecutorWebhook{}
			executor := &v1alpha.Executor{}
			assert.NoError(t, executorWebhook.ValidateUpdate(context.Background(), executor, executor))
		})
	}
}
func TestExecutorValidateCreate(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "executor validate update",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			executorWebhook := ExecutorWebhook{}
			executor := &v1alpha.Executor{}
			assert.NoError(t, executorWebhook.ValidateCreate(context.Background(), executor))
		})
	}
}
