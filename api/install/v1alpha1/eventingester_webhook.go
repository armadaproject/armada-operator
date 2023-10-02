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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var eventingesterlog = logf.Log.WithName("eventingester-resource")

func (r *EventIngester) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-install-armadaproject-io-v1alpha1-eventingester,mutating=true,failurePolicy=fail,sideEffects=None,groups=install.armadaproject.io,resources=eventingesters,verbs=create;update,versions=v1alpha1,name=meventingester.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &EventIngester{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *EventIngester) Default() {
	eventingesterlog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
	if r.Spec.Replicas == nil {
		r.Spec.Replicas = ptr.To[int32](1)
	}
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-install-armadaproject-io-v1alpha1-eventingester,mutating=false,failurePolicy=fail,sideEffects=None,groups=install.armadaproject.io,resources=eventingesters,verbs=create;update,versions=v1alpha1,name=veventingester.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &EventIngester{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *EventIngester) ValidateCreate() (admission.Warnings, error) {
	eventingesterlog.Info("validate create", "name", r.Name)

	// TODO(user): fill in your validation logic upon object creation.
	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *EventIngester) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	eventingesterlog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *EventIngester) ValidateDelete() (admission.Warnings, error) {
	eventingesterlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil, nil
}
