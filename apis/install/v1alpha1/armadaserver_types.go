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

// ArmadaServerSpec defines the desired state of ArmadaServer
type ArmadaServerSpec struct {
	// Labels is the map of labels which wil be added to all objects
	Labels map[string]string `json:"labels,omitempty"`
	// Image is the configuration block for the image repository and tag
	Image Image `json:"image"`
	// ApplicationConfig is the internal ArmadaServer configuration which will be created as a Kubernetes Secret and mounted in the Kubernetes Deployment object
	ApplicationConfig runtime.RawExtension `json:"applicationConfig"`
	// PrometheusConfig is the configuration block for Prometheus monitoring
	Prometheus *PrometheusConfig `json:"prometheus,omitempty"`
	// Resources is the configuration block for setting Executor resource requirements
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
	// Tolerations is the configuration block for specifying which taints the ArmadaServer pod can tolerate
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
	// TerminationGracePeriodSeconds specifies how many seconds should Kubernetes wait for the application to shut down gracefully before sending a KILL signal
	TerminationGracePeriodSeconds *int64 `json:"terminationGracePeriodSeconds,omitempty"`
	// if CustomServiceAccount is specified, then that service account is referenced in the Deployment (overrides service account defined in spec.serviceAccount field)
	CustomServiceAccount string `json:"customServiceAccount,omitempty"`
	// if ServiceAccount configuration is defined, it creates a new service account and references it in the deployment
	ServiceAccount *ServiceAccountConfig `json:"serviceAccount,omitempty"`
	Ingress        *IngressConfig        `json:"ingress,omitempty"`
	HostNames      []string              `json:"hostNames,omitempty"`
	ClusterIssuer  string                `json:"clusterIssuer"`
}

// ArmadaServerStatus defines the observed state of ArmadaServer
type ArmadaServerStatus struct {
}

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
