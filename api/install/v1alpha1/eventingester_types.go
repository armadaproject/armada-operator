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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EventIngesterSpec defines the desired state of EventIngester
type EventIngesterSpec struct {
	CommonSpecBase `json:""`

	// Replicas is the number of replicated instances for ArmadaServer
	Replicas *int32 `json:"replicas,omitempty"`
	// NodeSelector restricts the Executor pod to run on nodes matching the configured selectors
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
}

// EventIngesterStatus defines the observed state of EventIngester
type EventIngesterStatus struct{}

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
