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

var (
	_ KComponent     = (*KnativeEventing)(nil)
	_ KComponentSpec = (*KnativeEventingSpec)(nil)
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
func (ke *KnativeEventing) GetSpec() KComponentSpec {
	return &ke.Spec
}

// GetStatus implements KComponent
func (ke *KnativeEventing) GetStatus() KComponentStatus {
	return &ke.Status
}

// KnativeEventingSpec defines the desired state of KnativeEventing
type KnativeEventingSpec struct {
	CommonSpec `json:",inline"`

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
	// The default is `exclusion`.
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
	Awssqs     AwssqsSourceConfiguration     `json:"awssqs"`
	Ceph       CephSourceConfiguration       `json:"ceph"`
	Couchdb    CouchdbSourceConfiguration    `json:"couchdb"`
	Github     GithubSourceConfiguration     `json:"github"`
	Gitlab     GitlabSourceConfiguration     `json:"gitlab"`
	Kafka      KafkaSourceConfiguration      `json:"kafka"`
	Natss      NatssSourceConfiguration      `json:"natss"`
	Prometheus PrometheusSourceConfiguration `json:"prometheus"`
	Rabbitmq   RabbitmqSourceConfiguration   `json:"rabbitmq"`
	Redis      RedisSourceConfiguration      `json:"redis"`
}

// AwssqsSourceConfiguration specifies whether to enable the awssqs source.
type AwssqsSourceConfiguration struct {
	Enabled bool `json:"enabled"`
}

// CephSourceConfiguration specifies whether to enable the ceph source.
type CephSourceConfiguration struct {
	Enabled bool `json:"enabled"`
}

// CouchdbSourceConfiguration specifies whether to enable the couchdb source.
type CouchdbSourceConfiguration struct {
	Enabled bool `json:"enabled"`
}

// GithubSourceConfiguration specifies whether to enable the github source.
type GithubSourceConfiguration struct {
	Enabled bool `json:"enabled"`
}

// GitlabSourceConfiguration specifies whether to enable the gitlab source.
type GitlabSourceConfiguration struct {
	Enabled bool `json:"enabled"`
}

// KafkaSourceConfiguration specifies whether to enable the kafka source.
type KafkaSourceConfiguration struct {
	Enabled bool `json:"enabled"`
}

// NatssSourceConfiguration specifies whether to enable the natss source.
type NatssSourceConfiguration struct {
	Enabled bool `json:"enabled"`
}

// PrometheusSourceConfiguration specifies whether to enable the prometheus source.
type PrometheusSourceConfiguration struct {
	Enabled bool `json:"enabled"`
}

// RabbitmqSourceConfiguration specifies whether to enable the rabbitmq source.
type RabbitmqSourceConfiguration struct {
	Enabled bool `json:"enabled"`
}

// RedisSourceConfiguration specifies whether to enable the redis source.
type RedisSourceConfiguration struct {
	Enabled bool `json:"enabled"`
}
