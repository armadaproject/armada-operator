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
	Repository string `json:"repository"`
	Tag        string `json:"tag"`
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
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}
