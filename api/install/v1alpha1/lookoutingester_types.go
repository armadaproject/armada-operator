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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LookoutIngesterSpec defines the desired state of LookoutIngester
type LookoutIngesterSpec struct {
	CommonSpecBase `json:""`

	// Replicas is the number of replicated instances for LookoutIngester
	Replicas *int32 `json:"replicas,omitempty"`
}

// LookoutIngesterStatus defines the observed state of LookoutIngester
type LookoutIngesterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// LookoutIngester is the Schema for the lookoutingesters API
type LookoutIngester struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LookoutIngesterSpec   `json:"spec,omitempty"`
	Status LookoutIngesterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// LookoutIngesterList contains a list of LookoutIngester
type LookoutIngesterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LookoutIngester `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LookoutIngester{}, &LookoutIngesterList{})
}
