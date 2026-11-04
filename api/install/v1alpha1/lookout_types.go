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
	ClusterIssuer string `json:"clusterIssuer,omitempty"`
	// Migrate toggles whether to run migrations when installed
	Migrate *bool `json:"migrate,omitempty"`
	// MigrationDbWaitImage sets the image for the init container of the migration job.
	// The init container waits for the database and creates the database.
	// The image must contain the psql client and the nc command.
	// The default value is postgres:15.2-alpine.
	MigrationDbWaitImage *Image `json:"migrationDbWaitImage,omitempty"`
	// DbPruningEnabled when true a pruning CronJob is created
	DbPruningEnabled *bool `json:"dbPruningEnabled,omitempty"`
	// DbPruningSchedule schedule to use for db pruning CronJob
	DbPruningSchedule *string `json:"dbPruningSchedule,omitempty"`
	// SecurityContext defines the security options the container should be run with
	SecurityContext *corev1.SecurityContext `json:"securityContext,omitempty"`
	// PodSecurityContext defines the security options the pod should be run with
	PodSecurityContext *corev1.PodSecurityContext `json:"podSecurityContext,omitempty"`
	// ProfilingIngressConfig defines configuration for the profiling Ingress resource
	ProfilingIngressConfig *IngressConfig `json:"profilingIngressConfig,omitempty"`
	// PartitionJobTable configures the opt-in Job which converts the Lookout job table to its partitioned shape.
	// It requires Lookout v0.23.0 or later: earlier versions do not have the command it runs, so the Job fails without changing the database.
	PartitionJobTable *PartitionJobTableConfig `json:"partitionJobTable,omitempty"`
}

// PartitionJobTableConfig defines the settings of the Job which partitions the Lookout job table
type PartitionJobTableConfig struct {
	// Enabled when true creates a Job which partitions the Lookout job table, once the migration Job has succeeded.
	// The Job is not waited on and does not block the rest of the installation.
	// A finished Job is left in place and is not re-run; set this back to false once the table is partitioned.
	// Setting this to false does not delete an existing Job.
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`
	// ActiveDeadlineSeconds is the maximum duration in seconds the Job may run before it is terminated.
	// The default value is 86400 (24 hours).
	// +kubebuilder:validation:Minimum=1
	ActiveDeadlineSeconds *int64 `json:"activeDeadlineSeconds,omitempty"`
}

const (
	// ConditionTypeJobTablePartitioned reports the progress of the Job which partitions the Lookout job table
	ConditionTypeJobTablePartitioned = "JobTablePartitioned"
	// ReasonPartitionJobTableRunning means the Job which partitions the Lookout job table has not finished yet
	ReasonPartitionJobTableRunning = "Running"
	// ReasonPartitionJobTableSucceeded means the Lookout job table is partitioned
	ReasonPartitionJobTableSucceeded = "Succeeded"
	// ReasonPartitionJobTableFailed means the Job which partitions the Lookout job table has failed
	ReasonPartitionJobTableFailed = "Failed"
)

// LookoutStatus defines the observed state of lookout
type LookoutStatus struct {
	// Conditions represent the latest available observations of the Lookout's state
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

func init() {
	SchemeBuilder.Register(&Lookout{}, &LookoutList{})
}
