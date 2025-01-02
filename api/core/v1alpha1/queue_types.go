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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PermissionSubject struct {
	Kind string `json:"kind,omitempty"`
	Name string `json:"name,omitempty"`
}

type QueuePermissions struct {
	Subjects []PermissionSubject `json:"subjects,omitempty"`
	Verbs    []string            `json:"verbs,omitempty"`
}

// QueueSpec defines the desired state of Queue
type QueueSpec struct {
	// PriorityFactor is a multiplicative constant which is applied to the priority.
	PriorityFactor *resource.Quantity `json:"priorityFactor,omitempty"`
	// Permissions describe who can perform what operations on queue related resources.
	Permissions []QueuePermissions `json:"permissions,omitempty"`
}

// QueueStatus defines the observed state of Queue
type QueueStatus struct {
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Queue is the Schema for the queues API
type Queue struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   QueueSpec   `json:"spec,omitempty"`
	Status QueueStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// QueueList contains a list of Queue
type QueueList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Queue `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Queue{}, &QueueList{})
}
