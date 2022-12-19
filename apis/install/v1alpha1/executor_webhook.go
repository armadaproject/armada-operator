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
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

const (
	keySpec              = "spec"
	keyApplicationConfig = "applicationConfig"
	keyAPIConnection     = "apiConnection"
	keyArmadaURL         = "armadaUrl"
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
		r.Spec.Image.Repository = "gresearchdev/armada-executor"
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
	if r.Spec.Prometheus.ScrapeInterval == "" {
		r.Spec.Prometheus.ScrapeInterval = "10s"
	}
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-install-armadaproject-io-v1alpha1-executor,mutating=false,failurePolicy=fail,sideEffects=None,groups=install.armadaproject.io,resources=executors,verbs=create;update,versions=v1alpha1,name=vexecutor.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &Executor{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Executor) ValidateCreate() error {
	executorlog.Info("validate create", "name", r.Name)

	var allErrs field.ErrorList

	if err := validateApplicationConfig(nil, field.NewPath(keySpec).Child(keyApplicationConfig)); err != nil {
		allErrs = append(allErrs, err)
	}

	if len(allErrs) == 0 {
		return nil
	}

	return errors.NewInvalid(schema.GroupKind{Group: GroupVersion.Group, Kind: "Executor"}, r.Name, allErrs)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Executor) ValidateUpdate(old runtime.Object) error {
	executorlog.Info("validate update", "name", r.Name)

	var allErrs field.ErrorList

	if err := validateApplicationConfig(nil, field.NewPath(keySpec).Child(keyApplicationConfig)); err != nil {
		allErrs = append(allErrs, err)
	}

	if len(allErrs) == 0 {
		return nil
	}

	return errors.NewInvalid(schema.GroupKind{Group: GroupVersion.Group, Kind: "Executor"}, r.Name, allErrs)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Executor) ValidateDelete() error {
	executorlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}

func validateApplicationConfig(applicationConfig map[string]any, fldPath *field.Path) *field.Error {
	if applicationConfig == nil {
		return field.Invalid(fldPath, applicationConfig, "applicationConfig must be configured")
	}

	if applicationConfig[keyAPIConnection] == nil {
		return field.Invalid(fldPath.Child(keyAPIConnection), applicationConfig[keyAPIConnection], "apiConnection must be configured")
	}
	apiConnection, ok := applicationConfig[keyAPIConnection].(map[string]any)
	if !ok {
		return field.Invalid(
			fldPath.Child(keyAPIConnection),
			applicationConfig[keyAPIConnection],
			fmt.Sprintf("expected map[string]any, got %T", applicationConfig[keyAPIConnection]),
		)
	}

	if apiConnection[keyArmadaURL] == nil {
		return field.Invalid(fldPath.Child(keyAPIConnection).Child(keyArmadaURL), apiConnection["keyArmadaURL"], "armadaUrl must be provided")
	}

	_, ok = apiConnection[keyArmadaURL].(string)
	if !ok {
		return field.Invalid(
			fldPath.Child(keyAPIConnection).Child(keyArmadaURL),
			apiConnection[keyArmadaURL],
			fmt.Sprintf("expected string, got %T", apiConnection[keyArmadaURL]),
		)
	}

	return nil
}
