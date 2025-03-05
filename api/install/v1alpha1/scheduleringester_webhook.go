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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func (r *SchedulerIngester) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithDefaulter(&SchedulerIngesterDefaulter{}).
		WithValidator(&SchedulerIngesterValidator{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-install-armadaproject-io-v1alpha1-scheduleringester,mutating=true,failurePolicy=fail,sideEffects=None,groups=install.armadaproject.io,resources=scheduleringesters,verbs=create;update,versions=v1alpha1,name=mscheduleringester.kb.io,admissionReviewVersions=v1

var _ webhook.CustomDefaulter = &SchedulerIngesterDefaulter{}

type SchedulerIngesterDefaulter struct{}

func (d *SchedulerIngesterDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	schedulerIngester, ok := obj.(*SchedulerIngester)
	if !ok {
		return fmt.Errorf("expected a SchedulerIngester object in webhook but got %T", obj)
	}

	d.applyDefaults(schedulerIngester)
	return nil
}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (d *SchedulerIngesterDefaulter) applyDefaults(r *SchedulerIngester) {
	// image
	if r.Spec.Image.Repository == "" {
		r.Spec.Image.Repository = "gresearch/armada-scheduler-ingester"
	}

	if r.Spec.Replicas == nil {
		r.Spec.Replicas = ptr.To[int32](1)
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
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-install-armadaproject-io-v1alpha1-scheduleringester,mutating=false,failurePolicy=fail,sideEffects=None,groups=install.armadaproject.io,resources=scheduleringesters,verbs=create;update,versions=v1alpha1,name=vscheduleringester.kb.io,admissionReviewVersions=v1

var _ webhook.CustomValidator = &SchedulerIngesterValidator{}

type SchedulerIngesterValidator struct{}

func (s SchedulerIngesterValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	return nil, nil
}

func (s SchedulerIngesterValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (warnings admission.Warnings, err error) {
	return nil, nil
}

func (s SchedulerIngesterValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	return nil, nil
}
