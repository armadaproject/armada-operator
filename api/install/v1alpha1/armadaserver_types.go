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

// ArmadaServerSpec defines the desired state of ArmadaServer
type ArmadaServerSpec struct {
	CommonSpecBase `json:""`

	// Replicas is the number of replicated instances for ArmadaServer
	Replicas *int32 `json:"replicas,omitempty"`
	// NodeSelector restricts the ArmadaServer pod to run on nodes matching the configured selectors
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	// Ingress defines configuration for the Ingress resource
	Ingress *IngressConfig `json:"ingress,omitempty"`
	// ProfilingIngressConfig defines configuration for the profiling Ingress resource
	ProfilingIngressConfig *IngressConfig `json:"profilingIngressConfig,omitempty"`
	// An array of host names to build ingress rules for
	HostNames []string `json:"hostNames,omitempty"`
	// Who is issuing certificates for CA
	ClusterIssuer string `json:"clusterIssuer,omitempty"`
	// Run Pulsar Init Jobs On Startup
	PulsarInit bool `json:"pulsarInit,omitempty"`
	// SecurityContext defines the security options the container should be run with
	SecurityContext *corev1.SecurityContext `json:"securityContext,omitempty"`
	// PodSecurityContext defines the security options the pod should be run with
	PodSecurityContext *corev1.PodSecurityContext `json:"podSecurityContext,omitempty"`
}

// ArmadaServerStatus defines the observed state of ArmadaServer
type ArmadaServerStatus struct{}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ArmadaServer is the Schema for the Armada Server API
type ArmadaServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ArmadaServerSpec   `json:"spec,omitempty"`
	Status ArmadaServerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ArmadaServerList contains a list of ArmadaServer
type ArmadaServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ArmadaServer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ArmadaServer{}, &ArmadaServerList{})
}
