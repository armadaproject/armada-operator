package webhook

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	v1alpha "github.com/armadaproject/armada-operator/apis/install/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestBinocularsDefaultWebhook(t *testing.T) {
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
			webhook := BinocularsWebhook{}
			binoculars := binocularsSpec(tc.repository, tc.resources, tc.scrapeInterval)
			err := webhook.Default(context.Background(), binoculars)
			assert.NoError(t, err)
			if tc.repository != "" {
				assert.Equal(t, binoculars.Spec.Image.Repository, tc.repository)
			} else {
				assert.Equal(t, binoculars.Spec.Image.Repository, "gresearchdev/armada-binoculars")
			}
			if tc.resources != nil {
				assert.Equal(t, binoculars.Spec.Resources, tc.resources)
			} else {
				assert.Equal(t, binoculars.Spec.Resources, &corev1.ResourceRequirements{
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
				assert.Equal(t, binoculars.Spec.Prometheus.ScrapeInterval, tc.scrapeInterval)
			} else {
				assert.Equal(t, binoculars.Spec.Prometheus.ScrapeInterval, "10s")
			}
		})
	}
}

func binocularsSpec(repository string, resources *corev1.ResourceRequirements, scrapeInterval string) *v1alpha.Binoculars {
	binoculars := &v1alpha.Binoculars{
		Spec: v1alpha.BinocularsSpec{},
	}
	if repository != "" {
		binoculars.Spec.Image.Repository = repository
	}
	if resources != nil {
		binoculars.Spec.Resources = resources
	}
	if scrapeInterval != "" {
		binoculars.Spec.Prometheus.ScrapeInterval = scrapeInterval
	}
	return binoculars
}
