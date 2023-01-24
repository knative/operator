/*
Copyright 2022 The Knative Authors

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

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/operator/pkg/apis/operator/base"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

var (
	_ base.KComponent     = (*KnativeEventing)(nil)
	_ base.KComponentSpec = (*KnativeEventingSpec)(nil)
)

// KnativeEventing is the Schema for the eventings API
// +genclient
// +genreconciler:krshapedlogic=false
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type KnativeEventing struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KnativeEventingSpec   `json:"spec,omitempty"`
	Status KnativeEventingStatus `json:"status,omitempty"`
}

// GetSpec implements KComponent
func (ke *KnativeEventing) GetSpec() base.KComponentSpec {
	return &ke.Spec
}

// GetStatus implements KComponent
func (ke *KnativeEventing) GetStatus() base.KComponentStatus {
	return &ke.Status
}

// KnativeEventingSpec defines the desired state of KnativeEventing
type KnativeEventingSpec struct {
	base.CommonSpec `json:",inline"`

	// The default broker type to use for the brokers Knative creates.
	// If no value is provided, MTChannelBasedBroker will be used.
	// +optional
	DefaultBrokerClass string `json:"defaultBrokerClass,omitempty"`

	// SinkBindingSelectionMode specifies the NamespaceSelector and ObjectSelector
	// for the sinkbinding webhook.
	// If `inclusion` is selected, namespaces/objects labelled as `bindings.knative.dev/include:true`
	// will be considered by the sinkbinding webhook;
	// If `exclusion` is selected, namespaces/objects labelled as `bindings.knative.dev/exclude:true`
	// will NOT be considered by the sinkbinding webhook.
	// If no SINK_BINDING_SELECTION_MODE env var is given in the workloadOverrides for the
	// sinkinding webhook, the default `exclusion` is used.
	// +optional
	SinkBindingSelectionMode string `json:"sinkBindingSelectionMode,omitempty"`

	// Source allows configuration of different eventing sources to be shipped.
	// +optional
	Source *SourceConfigs `json:"source,omitempty"`
}

// KnativeEventingStatus defines the observed state of KnativeEventing
type KnativeEventingStatus struct {
	duckv1.Status `json:",inline"`

	// The version of the installed release
	// +optional
	Version string `json:"version,omitempty"`

	// The url links of the manifests, separated by comma
	// +optional
	Manifests []string `json:"manifests,omitempty"`
}

// KnativeEventingList contains a list of KnativeEventing
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type KnativeEventingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KnativeEventing `json:"items"`
}

// SourceConfigs specifies options for the eventing sources.
type SourceConfigs struct {
	Ceph     base.CephSourceConfiguration     `json:"ceph"`
	Github   base.GithubSourceConfiguration   `json:"github"`
	Gitlab   base.GitlabSourceConfiguration   `json:"gitlab"`
	Kafka    base.KafkaSourceConfiguration    `json:"kafka"`
	Rabbitmq base.RabbitmqSourceConfiguration `json:"rabbitmq"`
	Redis    base.RedisSourceConfiguration    `json:"redis"`
}
