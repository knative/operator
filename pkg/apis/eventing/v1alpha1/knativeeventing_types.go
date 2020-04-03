/*
Copyright 2019 The Knative Authors

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
	"knative.dev/pkg/apis"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KnativeEventing is the Schema for the eventings API
// +k8s:openapi-gen=true
type KnativeEventing struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KnativeEventingSpec   `json:"spec,omitempty"`
	Status KnativeEventingStatus `json:"status,omitempty"`
}

// Registry defines image overrides of knative images.
// This affects both apps/v1.Deployment and caching.internal.knative.dev/v1alpha1.Image.
// The default value is used as a default format to override for all knative deployments.
// The override values are specific to each knative deployment.
// +k8s:openapi-gen=true
type Registry struct {
	// The default image reference template to use for all knative images.
	// It takes the form of example-registry.io/custom/path/${NAME}:custom-tag
	// ${NAME} will be replaced by the deployment container name, or caching.internal.knative.dev/v1alpha1/Image name.
	// +optional
	Default string `json:"default,omitempty"`

	// A map of a container name or image name to the full image location of the individual knative image.
	// +optional
	Override map[string]string `json:"override,omitempty"`

	// A list of secrets to be used when pulling the knative images. The secret must be created in the
	// same namespace as the knative-eventing deployments, and not the namespace of this resource.
	// +optional
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
}

// KnativeEventingSpec defines the desired state of KnativeEventing
// +k8s:openapi-gen=true
type KnativeEventingSpec struct {
	// A means to override the corresponding deployment images in the upstream.
	// If no registry is provided, the knative release images will be used.
	// +optional
	Registry Registry `json:"registry,omitempty"`
}

// KnativeEventingStatus defines the observed state of KnativeEventing
// +k8s:openapi-gen=true
type KnativeEventingStatus struct {
	duckv1beta1.Status `json:",inline"`

	// The version of the installed release
	// +optional
	Version string `json:"version,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KnativeEventingList contains a list of KnativeEventing
type KnativeEventingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KnativeEventing `json:"items"`
}

const (
	// EventingConditionReady is set when the KnativeEventing Operator is installed, configured and ready.
	EventingConditionReady = apis.ConditionReady

	// InstallSucceeded is set when the Knative KnativeEventing is installed.
	InstallSucceeded apis.ConditionType = "InstallSucceeded"
)
