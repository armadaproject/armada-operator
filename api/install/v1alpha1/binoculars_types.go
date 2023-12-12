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

// Binoculars is the Schema for the binoculars API
type Binoculars struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BinocularsSpec   `json:"spec,omitempty"`
	Status BinocularsStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// BinocularsList contains a list of Binoculars
type BinocularsList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Binoculars `json:"items"`
}

// BinocularsSpec defines the desired state of Binoculars
type BinocularsSpec struct {
	CommonSpecBase `json:""`

	// Replicas is the number of replicated instances for Binoculars
	Replicas *int32 `json:"replicas"`
	// NodeSelector restricts the pod to run on nodes matching the configured selectors
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	// Ingress for this component. Used to inject labels/annotations into ingress
	Ingress *IngressConfig `json:"ingress,omitempty"`
	// An array of host names to build ingress rules for
	HostNames []string `json:"hostNames,omitempty"`
	// Who is issuing certificates for CA
	ClusterIssuer string `json:"clusterIssuer"`
	// SecurityContext defines the security options the container should be run with
	SecurityContext *corev1.SecurityContext `json:"securityContext,omitempty"`
	// PodSecurityContext defines the security options the pod should be run with
	PodSecurityContext *corev1.PodSecurityContext `json:"podSecurityContext,omitempty"`
}

// BinocularsStatus defines the observed state of binoculars
type BinocularsStatus struct{}

func init() {
	SchemeBuilder.Register(&Binoculars{}, &BinocularsList{})
}
