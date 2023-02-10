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
	// ScrapeInterval defines the interval at which Prometheus should scrape Executor metrics
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
	// Overide name for ingress
	NameOverride string `json:"nameOverride,omitempty"`
}

type AdditionalClusterRoleBinding struct {
	NameSuffix      string `json:"nameSuffix"`
	ClusterRoleName string `json:"clusterRoleName"`
}

// NOTE(clif): controller-gen does *not* handle embedded structs/promoted
// fields like one would hope. Perhaps the complication of json tags make this
// unreasonable, but it would've greatly simplified the definition of most of
// our service specs. Instead we're forced to resort to a lot more copy paste
// code across our service specs.
