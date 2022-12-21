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
	"github.com/armadaproject/armada-operator/apis/common"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// EventIngesterSpec defines the desired state of EventIngester
type EventIngesterSpec struct {
	// Name specifies the base name for all Kubernetes Resources
	Name string `json:"name"`
	// Labels is the map of labels which wil be added to all objects
	Labels map[string]string `json:"labels,omitempty"`
	// Image is the configuration block for the image repository and tag
	Image common.Image `json:"image"`
	// ApplicationConfig is the internal EventIngester configuration which will be created as a Kubernetes Secret and mounted in the Kubernetes Deployment object
	ApplicationConfig map[string]runtime.RawExtension `json:"applicationConfig"`
	// PrometheusConfig is the configuration block for Prometheus monitoring
	Prometheus *common.PrometheusConfig `json:"prometheus,omitempty"`
	// Resources is the configuration block for setting Executor resource requirements
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
	// Tolerations is the configuration block for specifying which taints the EventIngester pod can tolerate
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
	// TerminationGracePeriodSeconds specifies how many seconds should Kubernetes wait for the application to shut down gracefully before sending a KILL signal
	TerminationGracePeriodSeconds int `json:"terminationGracePeriodSeconds,omitempty"`
	// NodeSelector restricts the Executor pod to run on nodes matching the configured selectors
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	// if CustomServiceAccount is specified, then that service account is referenced in the Deployment (overrides service account defined in spec.serviceAccount field)
	CustomServiceAccount string `json:"customServiceAccount,omitempty"`
	// if ServiceAccount configuration is defined, it creates a new service account and references it in the deployment
	ServiceAccount common.ServiceAccountConfig `json:"serviceAccount,omitempty"`
}

// EventIngesterStatus defines the observed state of EventIngester
type EventIngesterStatus struct {
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// EventIngester is the Schema for the eventingesters API
type EventIngester struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EventIngesterSpec   `json:"spec,omitempty"`
	Status EventIngesterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// EventIngesterList contains a list of EventIngester
type EventIngesterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EventIngester `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EventIngester{}, &EventIngesterList{})
}
