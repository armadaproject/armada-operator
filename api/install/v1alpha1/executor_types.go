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
	schedulingv1 "k8s.io/api/scheduling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Executor is the Schema for the executors API
type Executor struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ExecutorSpec   `json:"spec,omitempty"`
	Status ExecutorStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ExecutorList contains a list of Executor
type ExecutorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Executor `json:"items"`
}

// ExecutorSpec defines the desired state of Executor
type ExecutorSpec struct {
	CommonSpecBase `json:""`

	// Replicas is the number of replicated instances for Executor
	Replicas *int32 `json:"replicas,omitempty"`
	// NodeSelector restricts the Executor pod to run on nodes matching the configured selectors
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	// Additional ClusterRoleBindings which will be created
	AdditionalClusterRoleBindings []AdditionalClusterRoleBinding `json:"additionalClusterRoleBindings,omitempty"`
	// List of PriorityClasses which will be created
	PriorityClasses []*schedulingv1.PriorityClass `json:"priorityClasses,omitempty"`
}

// ExecutorStatus defines the observed state of Executor
type ExecutorStatus struct{}

func init() {
	SchemeBuilder.Register(&Executor{}, &ExecutorList{})
}
