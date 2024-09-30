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

type Image struct {
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Pattern:="^([a-z0-9]+(?:[._-][a-z0-9]+)*/*)+$"
	Repository string `json:"repository"`
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Pattern:="^[a-zA-Z0-9_.-]*$"
	Tag string `json:"tag"`
}

type PrometheusConfig struct {
	// Enabled toggles should PrometheusRule and ServiceMonitor be created
	Enabled bool `json:"enabled,omitempty"`
	// Labels field enables adding additional labels to PrometheusRule and ServiceMonitor
	Labels map[string]string `json:"labels,omitempty"`
	// ScrapeInterval defines the interval at which Prometheus should scrape metrics
	// +kubebuilder:validation:Type:=string
	// +kubebuilder:validation:Format:=duration
	ScrapeInterval *metav1.Duration `json:"scrapeInterval,omitempty"`
}

type ServiceAccountConfig struct {
	Secrets                      []corev1.ObjectReference      `json:"secrets,omitempty"`
	ImagePullSecrets             []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
	AutomountServiceAccountToken *bool                         `json:"automountServiceAccountToken,omitempty"`
}

type IngressConfig struct {
	// Labels is the map of labels which wil be added to all objects
	Labels map[string]string `json:"labels,omitempty"`
	// Annotations is a map of annotations which will be added to all ingress rules
	Annotations map[string]string `json:"annotations,omitempty"`
	// The type of ingress that is used
	IngressClass string `json:"ingressClass,omitempty"`
	// An array of host names to build ingress rules for
	Hostnames []string `json:"hostNames,omitempty"`
	// Who is issuing certificates for CA
	ClusterIssuer string `json:"clusterIssuer,omitempty"`
}

type AdditionalClusterRoleBinding struct {
	NameSuffix      string `json:"nameSuffix"`
	ClusterRoleName string `json:"clusterRoleName"`
}

// CommonSpecBase is the common configuration for all services.
// NOTE(Clif): You must label this with `json:""` when using it as an embedded
// struct in order for controller-gen to use the promoted fields as expected.
type CommonSpecBase struct {
	// Labels is the map of labels which wil be added to all objects
	Labels map[string]string `json:"labels,omitempty"`
	// Image is the configuration block for the image repository and tag
	Image Image `json:"image"`
	// ApplicationConfig is the internal configuration of the application which will be created as a Kubernetes Secret and mounted in the Kubernetes Deployment object
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	ApplicationConfig runtime.RawExtension `json:"applicationConfig"`
	// PrometheusConfig is the configuration block for Prometheus monitoring
	Prometheus *PrometheusConfig `json:"prometheus,omitempty"`
	// Resources is the configuration block for setting resource requirements for this service
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
	// Tolerations is the configuration block for specifying which taints this pod can tolerate
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
	// TerminationGracePeriodSeconds specifies how many seconds should Kubernetes wait for the application to shut down gracefully before sending a KILL signal
	TerminationGracePeriodSeconds *int64 `json:"terminationGracePeriodSeconds,omitempty"`
	// if CustomServiceAccount is specified, then that service account is referenced in the Deployment (overrides service account defined in spec.serviceAccount field)
	CustomServiceAccount string `json:"customServiceAccount,omitempty"`
	// if ServiceAccount configuration is defined, it creates a new service account and references it in the deployment
	ServiceAccount *ServiceAccountConfig `json:"serviceAccount,omitempty"`
	// Extra environment variables that get added to deployment
	Environment []corev1.EnvVar `json:"environment,omitempty"`
	// Additional volumes that are mounted into deployments
	AdditionalVolumes []corev1.Volume `json:"additionalVolumes,omitempty"`
	// Additional volume mounts that are added as volumes
	AdditionalVolumeMounts []corev1.VolumeMount `json:"additionalVolumeMounts,omitempty"`
}

func GetDefaultSecurityContext() *corev1.SecurityContext {
	var runAsUser int64 = 1000
	var runAsGroup int64 = 2000
	allowPrivilegeEscalation := false
	return &corev1.SecurityContext{
		RunAsUser:                &runAsUser,
		RunAsGroup:               &runAsGroup,
		AllowPrivilegeEscalation: &allowPrivilegeEscalation,
	}
}

func GetDefaultPodSecurityContext() *corev1.PodSecurityContext {
	var runAsUser int64 = 1000
	var runAsGroup int64 = 2000
	return &corev1.PodSecurityContext{
		RunAsUser:  &runAsUser,
		RunAsGroup: &runAsGroup,
	}
}
