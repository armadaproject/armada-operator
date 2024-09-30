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
)

// SchedulerIngesterSpec defines the desired state of SchedulerIngester
type SchedulerIngesterSpec struct {
	CommonSpecBase `json:""`

	// Replicas is the number of replicated instances for SchedulerIngester
	Replicas *int32 `json:"replicas,omitempty"`
	// SecurityContext defines the security options the container should be run with
	SecurityContext *corev1.SecurityContext `json:"securityContext,omitempty"`
	// PodSecurityContext defines the security options the pod should be run with
	PodSecurityContext *corev1.PodSecurityContext `json:"podSecurityContext,omitempty"`
	// ProfilingIngressConfig defines configuration for the profiling Ingress resource
	ProfilingIngressConfig *IngressConfig `json:"profilingIngressConfig,omitempty"`
}

// SchedulerIngesterStatus defines the observed state of SchedulerIngester
type SchedulerIngesterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// SchedulerIngester is the Schema for the scheduleringesters API
type SchedulerIngester struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SchedulerIngesterSpec   `json:"spec,omitempty"`
	Status SchedulerIngesterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// SchedulerIngesterList contains a list of SchedulerIngester
type SchedulerIngesterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SchedulerIngester `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SchedulerIngester{}, &SchedulerIngesterList{})
}
