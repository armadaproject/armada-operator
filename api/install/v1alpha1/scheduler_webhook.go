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
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/runtime"

	"k8s.io/utils/ptr"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

func (r *Scheduler) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithDefaulter(&SchedulerDefaulter{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-install-armadaproject-io-v1alpha1-scheduler,mutating=true,failurePolicy=fail,sideEffects=None,groups=install.armadaproject.io,resources=schedulers,verbs=create;update,versions=v1alpha1,name=mscheduler.kb.io,admissionReviewVersions=v1

var _ webhook.CustomDefaulter = &SchedulerDefaulter{}

type SchedulerDefaulter struct{}

func (d *SchedulerDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	scheduler, ok := obj.(*Scheduler)
	if !ok {
		return fmt.Errorf("expected a Scheduler object in webhook but got %T", obj)
	}

	d.applyDefaults(scheduler)
	return nil
}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (d *SchedulerDefaulter) applyDefaults(r *Scheduler) {
	// image
	if r.Spec.Image.Repository == "" {
		r.Spec.Image.Repository = "gresearch/armada-scheduler"
	}

	if r.Spec.Migrate == nil {
		r.Spec.Migrate = ptr.To[bool](true)
	}

	// security context
	if r.Spec.SecurityContext == nil {
		r.Spec.SecurityContext = GetDefaultSecurityContext()
	}
	if r.Spec.PodSecurityContext == nil {
		r.Spec.PodSecurityContext = GetDefaultPodSecurityContext()
	}

	// resources
	if r.Spec.Resources == nil {
		r.Spec.Resources = &corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				"cpu":    resource.MustParse("300m"),
				"memory": resource.MustParse("1Gi"),
			},
			Requests: corev1.ResourceList{
				"cpu":    resource.MustParse("200m"),
				"memory": resource.MustParse("512Mi"),
			},
		}
	}

	if r.Spec.Pruner == nil {
		r.Spec.Pruner = &PrunerConfig{
			Schedule: "@hourly",
		}
		if r.Spec.Pruner.Resources == nil {
			r.Spec.Pruner.Resources = &corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					"cpu":    resource.MustParse("300m"),
					"memory": resource.MustParse("1Gi"),
				},
				Requests: corev1.ResourceList{
					"cpu":    resource.MustParse("200m"),
					"memory": resource.MustParse("512Mi"),
				},
			}
		}
	}

	// prometheus
	if r.Spec.Prometheus != nil && r.Spec.Prometheus.Enabled {
		if r.Spec.Prometheus.ScrapeInterval == nil {
			r.Spec.Prometheus.ScrapeInterval = &metav1.Duration{Duration: time.Second * 10}
		}
	}

	if r.Spec.CommonSpecBase.TopologyKey == "" {
		r.Spec.CommonSpecBase.TopologyKey = "kubernetes.io/hostname"
	}
}
