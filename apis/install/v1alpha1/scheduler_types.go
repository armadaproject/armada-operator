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

// Scheduler is the Schema for the scheduler API
type Scheduler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SchedulerSpec   `json:"spec,omitempty"`
	Status SchedulerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// SchedulerList contains a list of Scheduler
type SchedulerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Scheduler `json:"items"`
}

// SchedulerSpec defines the desired state of Scheduler
type SchedulerSpec struct {
	CommonSpecBase `json:""`

	// Replicas is the number of replicated instances
	Replicas int32 `json:"replicas,omitempty"`
	// Ingress defines labels and annotations for the Ingress controller of Scheduler
	Ingress *IngressConfig `json:"ingress,omitempty"`
	// An array of host names to build ingress rules for
	HostNames []string `json:"hostNames,omitempty"`
	// Who is issuing certificates for CA
	ClusterIssuer string `json:"clusterIssuer"`
	// Pruning config for cron job
	Pruner *PrunerConfig `json:"pruner,omitempty"`
}

// PrunerConfig definees the pruner cronjob settings
type PrunerConfig struct {
	Enabled   bool                         `json:"enabled"`
	Schedule  string                       `json:"schedule,omitempty"`
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
	Args      PrunerArgs                   `json:"args,omitempty"`
}

// PrunerArgs represent command-line args to the pruner cron job
type PrunerArgs struct {
	Timeout     string `json:"timeout,omitempty"`
	Batchsize   int32  `json:"batchsize,omitempty"`
	ExpireAfter string `json:"expireAfter,omitempty"`
}

// SchedulerStatus defines the observed state of scheduler
type SchedulerStatus struct{}

func init() {
	SchemeBuilder.Register(&Scheduler{}, &SchedulerList{})
}
