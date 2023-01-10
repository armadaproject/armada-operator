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

package webhook

import (
	"context"

	v1alpha "github.com/armadaproject/armada-operator/apis/install/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var eventingesterlog = logf.Log.WithName("eventingester-resource")

type EventIngesterWebhook struct{}

func SetupWebhookForEventIngester(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&v1alpha.EventIngester{}).
		WithDefaulter(&EventIngesterWebhook{}).
		WithValidator(&EventIngesterWebhook{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-install-armadaproject-io-v1alpha1-eventingester,mutating=true,failurePolicy=fail,sideEffects=None,groups=install.armadaproject.io,resources=eventingesters,verbs=create;update,versions=v1alpha1,name=meventingester.kb.io,admissionReviewVersions=v1

var _ webhook.CustomDefaulter = &EventIngesterWebhook{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *EventIngesterWebhook) Default(ctx context.Context, obj runtime.Object) error {
	eventIngester := obj.(*v1alpha.EventIngester)

	eventingesterlog.Info("default", "name", eventIngester.Name)
	// TODO(user): fill in your defaulting logic.
	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-install-armadaproject-io-v1alpha1-eventingester,mutating=false,failurePolicy=fail,sideEffects=None,groups=install.armadaproject.io,resources=eventingesters,verbs=create;update,versions=v1alpha1,name=veventingester.kb.io,admissionReviewVersions=v1

var _ webhook.CustomValidator = &EventIngesterWebhook{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *EventIngesterWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) error {
	eventIngester := obj.(*v1alpha.EventIngester)

	eventingesterlog.Info("validate create", "name", eventIngester.Name)

	// TODO(user): fill in your validation logic upon object creation.
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *EventIngesterWebhook) ValidateUpdate(ctx context.Context, obj, new runtime.Object) error {
	eventIngester := obj.(*v1alpha.EventIngester)

	eventingesterlog.Info("validate update", "name", eventIngester.Name)

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *EventIngesterWebhook) ValidateDelete(ctx context.Context, obj runtime.Object) error {
	eventIngester := obj.(*v1alpha.EventIngester)

	eventingesterlog.Info("validate delete", "name", eventIngester.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
