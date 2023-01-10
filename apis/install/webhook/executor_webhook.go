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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
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

type ExecutorWebhook struct{}

func SetupWebhookForExecutor(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&v1alpha.Executor{}).
		WithDefaulter(&ExecutorWebhook{}).
		WithValidator(&ExecutorWebhook{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-install-armadaproject-io-v1alpha1-executor,mutating=true,failurePolicy=fail,sideEffects=None,groups=install.armadaproject.io,resources=executors,verbs=create;update,versions=v1alpha1,name=mexecutor.kb.io,admissionReviewVersions=v1

var _ webhook.CustomDefaulter = &ExecutorWebhook{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (web *ExecutorWebhook) Default(ctx context.Context, obj runtime.Object) error {
	r := obj.(*v1alpha.Executor)

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
	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-install-armadaproject-io-v1alpha1-executor,mutating=false,failurePolicy=fail,sideEffects=None,groups=install.armadaproject.io,resources=executors,verbs=create;update,versions=v1alpha1,name=vexecutor.kb.io,admissionReviewVersions=v1

var _ webhook.CustomValidator = &ExecutorWebhook{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (web *ExecutorWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) error {
	r := obj.(*v1alpha.Executor)

	executorlog.Info("validate create", "name", r.Name)

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (web *ExecutorWebhook) ValidateUpdate(ctx context.Context, obj, old runtime.Object) error {
	r := obj.(*v1alpha.Executor)
	executorlog.Info("validate update", "name", r.Name)

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (web *ExecutorWebhook) ValidateDelete(ctx context.Context, obj runtime.Object) error {
	r := obj.(*v1alpha.Executor)

	executorlog.Info("validate delete", "name", r.Name)

	return nil
}
