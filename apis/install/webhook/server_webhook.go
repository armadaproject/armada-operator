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
var armadaserverlog = logf.Log.WithName("armadaserver-resource")

type ArmadaServerWebhook struct{}

func SetupWebhookForArmadaServer(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&v1alpha.Server{}).
		WithDefaulter(&ArmadaServerWebhook{}).
		WithValidator(&ArmadaServerWebhook{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-install-armadaproject-io-v1alpha1-server,mutating=true,failurePolicy=fail,sideEffects=None,groups=install.armadaproject.io,resources=server,verbs=create;update,versions=v1alpha1,name=mserver.kb.io,admissionReviewVersions=v1

var _ webhook.CustomDefaulter = &ArmadaServerWebhook{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *ArmadaServerWebhook) Default(ctx context.Context, obj runtime.Object) error {
	eventIngester := obj.(*v1alpha.Server)

	armadaserverlog.Info("default", "name", eventIngester.Name)
	// TODO(user): fill in your defaulting logic.
	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-install-armadaproject-io-v1alpha1-server,mutating=false,failurePolicy=fail,sideEffects=None,groups=install.armadaproject.io,resources=server,verbs=create;update,versions=v1alpha1,name=vserver.kb.io,admissionReviewVersions=v1

var _ webhook.CustomValidator = &ArmadaServerWebhook{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *ArmadaServerWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) error {
	server := obj.(*v1alpha.Server)

	armadaserverlog.Info("validate create", "name", server.Name)

	// TODO(user): fill in your validation logic upon object creation.
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *ArmadaServerWebhook) ValidateUpdate(ctx context.Context, obj, new runtime.Object) error {
	server := obj.(*v1alpha.Server)

	armadaserverlog.Info("validate update", "name", server.Name)

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *ArmadaServerWebhook) ValidateDelete(ctx context.Context, obj runtime.Object) error {
	server := obj.(*v1alpha.Server)

	armadaserverlog.Info("validate delete", "name", server.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
