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
	schedulingv1 "k8s.io/api/scheduling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
	// Labels is the map of labels which wil be added to all objects
	Labels map[string]string `json:"labels,omitempty"`
	// Image is the configuration block for the image repository and tag
	Image Image `json:"image"`
	// ApplicationConfig is the internal Executor configuration which will be created as a Kubernetes Secret and mounted in the Kubernetes Deployment object
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	ApplicationConfig runtime.RawExtension `json:"applicationConfig"`
	// PrometheusConfig is the configuration block for Prometheus monitoring
	Prometheus *PrometheusConfig `json:"prometheus,omitempty"`
	// Resources is the configuration block for setting Executor resource requirements
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
	// Tolerations is the configuration block for specifying which taints can the Executor pod tolerate
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
	// TerminationGracePeriodSeconds specifies how many seconds should Kubernetes wait for the application to shut down gracefully before sending a KILL signal
	TerminationGracePeriodSeconds *int64 `json:"terminationGracePeriodSeconds,omitempty"`
	// NodeSelector restricts the Executor pod to run on nodes matching the configured selectors
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	// if CustomServiceAccount is specified,
	// then the custom service account is referenced in the Deployment
	// and this will override service account defined in spec.serviceAccount field
	CustomServiceAccount string `json:"customServiceAccount,omitempty"`
	// if ServiceAccount configuration is defined, it creates a new service account and references it in the deployment
	ServiceAccount *ServiceAccountConfig `json:"serviceAccount,omitempty"`
	// Extra environment variables that get added to deployment
	Environment []corev1.EnvVar `json:"env,omitempty"`
	// Additional ClusterRoleBindings which will be created
	AdditionalClusterRoleBindings []AdditionalClusterRoleBinding `json:"additionalClusterRoleBindings,omitempty"`
	// Additional VolumeMounts which will be attached to the app container
	AdditionalVolumeMounts []corev1.VolumeMount `json:"additionalVolumeMounts,omitempty"`
	// Additional Volumes which will be added to the Deployment
	AdditionalVolumes []corev1.Volume `json:"additionalVolumes,omitempty"`
	// List of PriorityClasses which will be created
	PriorityClasses []*schedulingv1.PriorityClass `json:"priorityClasses,omitempty"`
}

// ExecutorStatus defines the observed state of Executor
type ExecutorStatus struct{}

func init() {
	SchemeBuilder.Register(&Executor{}, &ExecutorList{})
}
