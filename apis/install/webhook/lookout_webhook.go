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
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

type LookoutWebhook struct{}

func SetupWebhookForLookout(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&v1alpha.Lookout{}).
		WithDefaulter(&LookoutWebhook{}).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-install-armadaproject-io-v1alpha1-lookout,mutating=true,failurePolicy=fail,sideEffects=None,groups=install.armadaproject.io,resources=lookout,verbs=create;update,versions=v1alpha1,name=mlookout.kb.io,admissionReviewVersions=v1

var _ webhook.CustomDefaulter = &LookoutWebhook{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *LookoutWebhook) Default(ctx context.Context, obj runtime.Object) error {
	lookout := obj.(*v1alpha.Lookout)
	executorlog.Info("default", "name", lookout.Name)
	lookout.Spec = setLookoutDefaults(lookout).Spec
	return nil
}

func setLookoutDefaults(r *v1alpha.Lookout) *v1alpha.Lookout {
	// image
	if r.Spec.Image.Repository == "" {
		r.Spec.Image.Repository = "gresearchdev/armada-lookout"
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

	return r
}
