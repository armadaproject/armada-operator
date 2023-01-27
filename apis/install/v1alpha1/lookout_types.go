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
	"k8s.io/apimachinery/pkg/runtime"
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
	// MigrateDatabase must be true to enable database migration job
	MigrateDatabase bool `json:"migrateDatabase"`
	// Set enableV2 to true to enable the new version of Lookout
	EnableV2 bool `json:"enableV2"`
	// Replicas is the number of replicated instances for ArmadaServer
	Replicas int32 `json:"replicas,omitempty"`
	// Labels is the map of labels which wil be added to all objects
	Labels map[string]string `json:"labels,omitempty"`
	// Image is the configuration block for the image repository and tag
	Image Image `json:"image"`
	// ApplicationConfig is the internal Lookout configuration which will be created as a Kubernetes Secret and mounted in the Kubernetes Deployment object
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	ApplicationConfig runtime.RawExtension `json:"applicationConfig"`
	// PrometheusConfig is the configuration block for Prometheus monitoring
	Prometheus PrometheusConfig `json:"prometheus,omitempty"`
	// Resources is the configuration block for setting Lookout resource requirements
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
	// Tolerations is the configuration block for specifying which taints can the Lookout pod tolerate
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
	// TerminationGracePeriodSeconds specifies how many seconds should Kubernetes wait for the application to shut down gracefully before sending a KILL signal
	TerminationGracePeriodSeconds int `json:"terminationGracePeriodSeconds,omitempty"`
	// NodeSelector restricts the Lookout pod to run on nodes matching the configured selectors
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	// if CustomServiceAccount is specified, then that service account is referenced in the Deployment (overrides service account defined in spec.serviceAccount field)
	CustomServiceAccount string `json:"customServiceAccount,omitempty"`
	// if ServiceAccount configuration is defined, it creates a new service account and references it in the deployment
	ServiceAccount *ServiceAccountConfig `json:"serviceAccount,omitempty"`
	Ingress        *IngressConfig        `json:"ingress,omitempty"`
	// An array of host names to build ingress rules for
	HostNames []string `json:"hostNames,omitempty"`
	// Who is issuing certificates for CA
	ClusterIssuer string `json:"clusterIssuer"`
	// Extra environment variables that get added to deployment
	Environment []Environment `json:"environment,omitempty"`
	// Additional volumes that are mounted into deployments
	AdditionalVolumes []AdditionalVolume `json:"additionalVolumes,omitempty"`
	// Additional volume mounts that are added as volumes
	AdditionalVolumeMounts []AdditionalVolumeMounts `json:"additionalVolumeMounts,omitempty"`
}

// LookoutStatus defines the observed state of lookout
type LookoutStatus struct{}

func init() {
	SchemeBuilder.Register(&Lookout{}, &LookoutList{})
}
