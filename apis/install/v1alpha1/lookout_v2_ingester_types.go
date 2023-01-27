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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// TODO: Clif - should this just look like the other services or is there unique
// functionality?
// LookoutV2IngesterSpec defines the desired state of LookoutV2Ingester
// TODO: Should we be using OpenAPI validation markers on our Specs?
// See: https://sdk.operatorframework.io/docs/building-operators/golang/tutorial/#openapi-validation
type LookoutV2IngesterSpec struct {
	// Labels is the map of labels which wil be added to all objects
	Labels map[string]string `json:"labels,omitempty"`
	// Image is the configuration block for the image repository and tag
	Image Image `json:"image"`
	// ApplicationConfig is the internal LookoutV2Ingester configuration which will be created as a Kubernetes Secret and mounted in the Kubernetes Deployment object
	ApplicationConfig runtime.RawExtension `json:"applicationConfig"`
	// PrometheusConfig is the configuration block for Prometheus monitoring
	Prometheus *PrometheusConfig `json:"prometheus,omitempty"`
	// Resources is the configuration block for setting LookoutV2Ingester resource requirements
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
	// Tolerations is the configuration block for specifying which taints can the LookoutV2Ingester pod tolerate
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
	// TerminationGracePeriodSeconds specifies how many seconds should Kubernetes wait for the application to shut down gracefully before sending a KILL signal
	TerminationGracePeriodSeconds *int64 `json:"terminationGracePeriodSeconds,omitempty"`
	// if CustomServiceAccount is specified, then that service account is referenced in the Deployment (overrides service account defined in spec.serviceAccount field)
	CustomServiceAccount string `json:"customServiceAccount,omitempty"`
	// if ServiceAccount configuration is defined, it creates a new service account and references it in the deployment
	ServiceAccount *ServiceAccountConfig `json:"serviceAccount,omitempty"`
}

// LookoutV2IngesterStatus defines the observed state of LookoutV2Ingester
type LookoutV2IngesterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// LookoutV2Ingester is the Schema for the lookoutV2ingesters API
type LookoutV2Ingester struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LookoutV2IngesterSpec   `json:"spec,omitempty"`
	Status LookoutV2IngesterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// LookoutV2IngesterList contains a list of LookoutV2Ingester
type LookoutV2IngesterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LookoutV2Ingester `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LookoutV2Ingester{}, &LookoutV2IngesterList{})
}
