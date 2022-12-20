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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Lookout is the Schema for the lookout API
type Lookout struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LookoutSpec   `json:"spec,omitempty"`
	Status LookoutStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// LookoutList contains a list of Lookout
type LookoutList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Lookout `json:"items"`
}

// LookoutSpec defines the desired state of Lookout
type LookoutSpec struct {
	// Name specifies the base name for all Kubernetes Resources
	Name string `json:"name"`
	// Labels is the map of labels which wil be added to all objects
	Labels map[string]string `json:"labels,omitempty"`
	// Image is the configuration block for the image repository and tag
	Image common.Image `json:"image"`
	// ApplicationConfig is the internal Lookout configuration which will be created as a Kubernetes Secret and mounted in the Kubernetes Deployment object
	ApplicationConfig map[string]runtime.RawExtension `json:"applicationConfig"`
	// Strategy is the configuration block for the Kubernetes Deployment Strategy
	strategy map[string]runtime.RawExtension `json:"strategy"`
	// PrometheusConfig is the configuration block for Prometheus monitoring
	Prometheus PrometheusConfig `json:"prometheus,omitempty"`
	// Resources is the configuration block for setting Lookout resource requirements
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
	// Replicas is the number of replicas for the Lookout Deployment
	Replicas int `json:"replicas,omitempty"`
	// TerminationGracePeriodSeconds specifies how many seconds should Kubernetes wait for the application to shut down gracefully before sending a KILL signal
	TerminationGracePeriodSeconds int `json:"terminationGracePeriodSeconds,omitempty"`
	// if CustomServiceAccount is specified, then that service account is referenced in the Deployment (overrides service account defined in spec.serviceAccount field)
	CustomServiceAccount string `json:"customServiceAccount,omitempty"`
	// if ServiceAccount configuration is defined, it creates a new service account and references it in the deployment
	ServiceAccount *ServiceAccountConfig `json:"serviceAccount,omitempty"`
}

type PrometheusConfig struct {
	// Enabled toggles should PrometheusRule and ServiceMonitor be created
	Enabled bool `json:"enabled,omitempty"`
	// Labels field enables adding additional labels to PrometheusRule and ServiceMonitor
	Labels map[string]string `json:"labels,omitempty"`
	// ScrapeInterval defines the interval at which Prometheus should scrape Lookout metrics
	ScrapeInterval string `json:"scrapeInterval,omitempty"`
}

type ServiceAccountConfig struct {
	Secrets                      []corev1.ObjectReference      `json:"secrets,omitempty"`
	ImagePullSecrets             []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
	AutomountServiceAccountToken *bool                         `json:"automountServiceAccountToken,omitempty"`
}

// LookoutStatus defines the observed state of Lookout
type LookoutStatus struct {
}

func init() {
	SchemeBuilder.Register(&Lookout{}, &LookoutList{})
}