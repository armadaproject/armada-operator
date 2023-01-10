package webhook

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	v1alpha "github.com/armadaproject/armada-operator/apis/install/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestLookoutDefaultWebhook(t *testing.T) {
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
			webhook := LookoutWebhook{}
			lookout := lookoutSpec(tc.repository, tc.resources, tc.scrapeInterval)
			err := webhook.Default(context.Background(), lookout)
			assert.NoError(t, err)
			if tc.repository != "" {
				assert.Equal(t, lookout.Spec.Image.Repository, tc.repository)
			} else {
				assert.Equal(t, lookout.Spec.Image.Repository, "gresearchdev/armada-lookout")
			}
			if tc.resources != nil {
				assert.Equal(t, lookout.Spec.Resources, tc.resources)
			} else {
				assert.Equal(t, lookout.Spec.Resources, &corev1.ResourceRequirements{
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
				assert.Equal(t, lookout.Spec.Prometheus.ScrapeInterval, tc.scrapeInterval)
			} else {
				assert.Equal(t, lookout.Spec.Prometheus.ScrapeInterval, "10s")
			}
		})
	}
}

func lookoutSpec(repository string, resources *corev1.ResourceRequirements, scrapeInterval string) *v1alpha.Lookout {
	lookout := &v1alpha.Lookout{
		Spec: v1alpha.LookoutSpec{},
	}
	if repository != "" {
		lookout.Spec.Image.Repository = repository
	}
	if resources != nil {
		lookout.Spec.Resources = resources
	}
	if scrapeInterval != "" {
		lookout.Spec.Prometheus.ScrapeInterval = scrapeInterval
	}
	return lookout
}
