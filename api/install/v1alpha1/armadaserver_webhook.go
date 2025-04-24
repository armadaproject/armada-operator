/*
Copyright 2022.

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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func (r *ArmadaServer) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithDefaulter(&ArmadaServerDefaulter{}).
		WithValidator(&ArmadaServerValidator{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-install-armadaproject-io-v1alpha1-armadaserver,mutating=true,failurePolicy=fail,sideEffects=None,groups=install.armadaproject.io,resources=armadaservers,verbs=create;update,versions=v1alpha1,name=armadaserver.kb.io,admissionReviewVersions=v1

var _ webhook.CustomDefaulter = &ArmadaServerDefaulter{}

type ArmadaServerDefaulter struct{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (d *ArmadaServerDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	armadaServer, ok := obj.(*ArmadaServer)
	if !ok {
		return fmt.Errorf("expected an ArmadaServer object in webhook but got %T", obj)
	}

	// Set default values
	d.applyDefaults(armadaServer)
	return nil
}

func (d *ArmadaServerDefaulter) applyDefaults(r *ArmadaServer) {
	// image
	if r.Spec.Image.Repository == "" {
		r.Spec.Image.Repository = "gresearch/armada-server"
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

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-install-armadaproject-io-v1alpha1-armadaserver,mutating=true,failurePolicy=fail,sideEffects=None,groups=install.armadaproject.io,resources=armadaservers,verbs=create;update,versions=v1alpha1,name=varmadaserver.kb.io,admissionReviewVersions=v1

var _ webhook.CustomValidator = &ArmadaServerValidator{}

type ArmadaServerValidator struct{}

func (v ArmadaServerValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	// TODO(user): fill in your validation logic upon object creation.
	return nil, nil
}

func (v ArmadaServerValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (warnings admission.Warnings, err error) {
	// TODO(user): fill in your validation logic upon object creation.
	return nil, nil
}

func (v ArmadaServerValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	// TODO(user): fill in your validation logic upon object creation.
	return nil, nil
}
