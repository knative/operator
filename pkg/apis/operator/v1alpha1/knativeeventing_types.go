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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KnativeEventing is the Schema for the eventings API
// +k8s:openapi-gen=true
type KnativeEventing struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KnativeEventingSpec   `json:"spec,omitempty"`
	Status KnativeEventingStatus `json:"status,omitempty"`
}

// KnativeEventingSpec defines the desired state of KnativeEventing
// +k8s:openapi-gen=true
type KnativeEventingSpec struct {
	CommonSpec `json:",inline"`

	// The default broker type to use for the brokers Knative creates.
	// If no value is provided, ChannelBasedBroker will be used.
	// +optional
	DefaultBrokerClass string `json:"defaultBrokerClass,omitempty"`
}

// KnativeEventingStatus defines the observed state of KnativeEventing
// +k8s:openapi-gen=true
type KnativeEventingStatus struct {
	duckv1.Status `json:",inline"`

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
