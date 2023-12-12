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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	CommonSpecBase `json:""`

	// Replicas is the number of replicated instances for Lookout
	Replicas *int32 `json:"replicas,omitempty"`
	// NodeSelector restricts the Lookout pod to run on nodes matching the configured selectors
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	// Ingress defines labels and annotations for the Ingress controller of Lookout
	Ingress *IngressConfig `json:"ingress,omitempty"`
	// An array of host names to build ingress rules for
	HostNames []string `json:"hostNames,omitempty"`
	// Who is issuing certificates for CA
	ClusterIssuer string `json:"clusterIssuer"`
	// Migrate toggles whether to run migrations when installed
	Migrate *bool `json:"migrate,omitempty"`
	// DbPruningEnabled when true a pruning CronJob is created
	DbPruningEnabled *bool `json:"dbPruningEnabled,omitempty"`
	// DbPruningSchedule schedule to use for db pruning CronJob
	DbPruningSchedule *string `json:"dbPruningSchedule,omitempty"`
	// SecurityContext defines the security options the container should be run with
	SecurityContext *corev1.SecurityContext `json:"securityContext,omitempty"`
	// PodSecurityContext defines the security options the pod should be run with
	PodSecurityContext *corev1.PodSecurityContext `json:"podSecurityContext,omitempty"`
}

// LookoutStatus defines the observed state of lookout
type LookoutStatus struct{}

func init() {
	SchemeBuilder.Register(&Lookout{}, &LookoutList{})
}
