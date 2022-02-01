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
	"knative.dev/operator/pkg/apis/operator/base"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

var (
	_ base.KComponent     = (*KnativeServing)(nil)
	_ base.KComponentSpec = (*KnativeServingSpec)(nil)
)

// KnativeServing is the Schema for the knativeservings API
// +genclient
// +genreconciler:krshapedlogic=false
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type KnativeServing struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KnativeServingSpec   `json:"spec,omitempty"`
	Status KnativeServingStatus `json:"status,omitempty"`
}

// GetSpec implements KComponent
func (ks *KnativeServing) GetSpec() base.KComponentSpec {
	return &ks.Spec
}

// GetStatus implements KComponent
func (ks *KnativeServing) GetStatus() base.KComponentStatus {
	return &ks.Status
}

// KnativeServingSpec defines the desired state of KnativeServing
type KnativeServingSpec struct {
	base.CommonSpec `json:",inline"`

	// DEPRECATED.
	// DeprecatedKnativeIngressGateway is to override the knative-ingress-gateway.
	// +optional
	DeprecatedKnativeIngressGateway base.IstioGatewayOverride `json:"knative-ingress-gateway,omitempty"`

	// DEPRECATED.
	// DeprecatedClusterLocalGateway is to override the cluster-local-gateway.
	// +optional
	DeprecatedClusterLocalGateway base.IstioGatewayOverride `json:"cluster-local-gateway,omitempty"`

	// Enables controller to trust registries with self-signed certificates
	ControllerCustomCerts base.CustomCerts `json:"controller-custom-certs,omitempty"`

	// Ingress allows configuration of different ingress adapters to be shipped.
	Ingress *IngressConfigs `json:"ingress,omitempty"`
}

// KnativeServingStatus defines the observed state of KnativeServing
type KnativeServingStatus struct {
	duckv1.Status `json:",inline"`

	// The version of the installed release
	// +optional
	Version string `json:"version,omitempty"`

	// The url links of the manifests, separated by comma
	// +optional
	Manifests []string `json:"manifests,omitempty"`
}

// KnativeServingList contains a list of KnativeServing
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type KnativeServingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KnativeServing `json:"items"`
}

// IngressConfigs specifies options for the ingresses.
type IngressConfigs struct {
	Istio   base.IstioIngressConfiguration   `json:"istio"`
	Kourier base.KourierIngressConfiguration `json:"kourier"`
	Contour base.ContourIngressConfiguration `json:"contour"`
}
