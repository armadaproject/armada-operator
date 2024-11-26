/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/utils/ptr"
)

func TestScheduler_Default(t *testing.T) {
	tests := []struct {
		name   string
		input  *Scheduler
		assert func(*testing.T, *Scheduler)
	}{
		{
			name: "field is set and not overridden",
			input: &Scheduler{
				Spec: SchedulerSpec{
					CommonSpecBase: CommonSpecBase{
						Image: Image{
							Repository: "test/armada-scheduler",
							Tag:        "latest",
						},
					},
				},
			},
			assert: func(t *testing.T, s *Scheduler) {
				assert.Equal(t, "test/armada-scheduler", s.Spec.Image.Repository)
			},
		},
		{
			name: "field is not set and gets defaulted",
			input: &Scheduler{
				Spec: SchedulerSpec{
					Migrate: nil,
				},
			},
			assert: func(t *testing.T, s *Scheduler) {
				assert.Equal(t, ptr.To(true), s.Spec.Migrate)
			},
		},
		{
			name: "nothing is set and everything gets defaulted",
			input: &Scheduler{
				Spec: SchedulerSpec{},
			},
			assert: func(t *testing.T, scheduler *Scheduler) {
				assert.Equal(t, "gresearch/armada-scheduler", scheduler.Spec.Image.Repository)
				assert.Equal(t, ptr.To[bool](true), scheduler.Spec.Migrate)
				assert.Equal(t, GetDefaultSecurityContext(), scheduler.Spec.SecurityContext)
				assert.Equal(t, GetDefaultPodSecurityContext(), scheduler.Spec.PodSecurityContext)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.input.Default()
			tt.assert(t, tt.input)
		})
	}
}
