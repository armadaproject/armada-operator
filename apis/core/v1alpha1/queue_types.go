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

type Subject struct {
	Name string `json:"name,omitempty"`
	Kind string `json:"kind,omitempty"`
}

type Permission struct {
	Subjects []Subject `json:"subjects,omitempty"`
	Verbs    []string  `json:"verbs,omitempty"`
}

// QueueSpec defines the desired state of Queue
type QueueSpec struct {
	//Priority for queue
	PriorityFactor string `json:"priorityFactor"`
	// Owners of queue
	UserOwners []string `json:"userOwners,omitempty"`
	// Group owners of queue
	GroupOwners []string `json:"groupOwners,omitempty"`
	// An array of permissions for this queue
	Permission []Permission `json:"permissions,omitempty"`
	// Resource requirements for queue
	ResourceLimits map[string]string `json:"resourceLimits,omitempty"`
}

// QueueStatus defines the observed state of Queue
type QueueStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
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
