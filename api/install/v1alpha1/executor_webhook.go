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
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var executorlog = logf.Log.WithName("executor-resource")

func (r *Executor) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-install-armadaproject-io-v1alpha1-executor,mutating=true,failurePolicy=fail,sideEffects=None,groups=install.armadaproject.io,resources=executors,verbs=create;update,versions=v1alpha1,name=mexecutor.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &Executor{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Executor) Default() {
	executorlog.Info("default", "name", r.Name)

	// image
	if r.Spec.Image.Repository == "" {
		r.Spec.Image.Repository = "gresearch/armada-executor"
	}

	if r.Spec.Replicas == nil {
		r.Spec.Replicas = pointer.Int32(1)
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
	if r.Spec.Prometheus != nil {
		if r.Spec.Prometheus.ScrapeInterval == nil {
			r.Spec.Prometheus.ScrapeInterval = &metav1.Duration{Duration: time.Second * 10}
		}
	}
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-install-armadaproject-io-v1alpha1-executor,mutating=false,failurePolicy=fail,sideEffects=None,groups=install.armadaproject.io,resources=executors,verbs=create;update,versions=v1alpha1,name=vexecutor.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &Executor{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Executor) ValidateCreate() (admission.Warnings, error) {
	executorlog.Info("validate create", "name", r.Name)

	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Executor) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	executorlog.Info("validate update", "name", r.Name)

	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Executor) ValidateDelete() (admission.Warnings, error) {
	executorlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil, nil
}
