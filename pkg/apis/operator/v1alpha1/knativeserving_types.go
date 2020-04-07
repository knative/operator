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
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

const (
	DependenciesInstalled apis.ConditionType = "DependenciesInstalled"
	InstallSucceeded      apis.ConditionType = "InstallSucceeded"
	DeploymentsAvailable  apis.ConditionType = "DeploymentsAvailable"
)

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
	// same namespace as the knative-serving deployments, and not the namespace of this resource.
	// +optional
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
}

// IstioGatewayOverride override the knative-ingress-gateway and cluster-local-gateway
type IstioGatewayOverride struct {
	// A map of values to replace the "selector" values in the knative-ingress-gateway and cluster-local-gateway
	Selector map[string]string `json:"selector,omitempty"`
}

// CustomCerts refers to either a ConfigMap or Secret containing valid
// CA certificates
type CustomCerts struct {
	// One of ConfigMap or Secret
	Type string `json:"type"`

	// The name of the ConfigMap or Secret
	Name string `json:"name"`
}

// HighAvailability specifies options for deploying Knative Serving control
// plane in a highly available manner. Note that HighAvailability is still in
// progress and does not currently provide a completely HA control plane.
type HighAvailability struct {
	// Replicas is the number of replicas that HA parts of the control plane
	// will be scaled to.
	Replicas int32 `json:"replicas"`
}

// ResourceRequirementsOverride enables the user to override any container's
// resource requests/limits specified in the embedded manifest
type ResourceRequirementsOverride struct {
	// The container name
	Container string `json:"container"`
	// The desired ResourceRequirements
	corev1.ResourceRequirements
}

// KnativeServingSpec defines the desired state of KnativeServing
// +k8s:openapi-gen=true
type KnativeServingSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html

	// A means to override the corresponding entries in the upstream configmaps
	// +optional
	Config map[string]map[string]string `json:"config,omitempty"`

	// A means to override the corresponding deployment images in the upstream.
	// If no registry is provided, the knative release images will be used.
	// +optional
	Registry Registry `json:"registry,omitempty"`

	// A means to override the knative-ingress-gateway
	KnativeIngressGateway IstioGatewayOverride `json:"knative-ingress-gateway,omitempty"`

	// A means to override the cluster-local-gateway
	ClusterLocalGateway IstioGatewayOverride `json:"cluster-local-gateway,omitempty"`

	// Enables controller to trust registries with self-signed certificates
	ControllerCustomCerts CustomCerts `json:"controller-custom-certs,omitempty"`

	// Allows specification of HA control plane
	// +optional
	HighAvailability *HighAvailability `json:"high-availability,omitempty"`

	// Override containers' resource requirements
	// +optional
	Resources []ResourceRequirementsOverride `json:"resources,omitempty"`
}

// KnativeServingStatus defines the observed state of KnativeServing
// +k8s:openapi-gen=true
type KnativeServingStatus struct {
	duckv1.Status `json:",inline"`

	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags:
	// https://book.kubebuilder.io/beyond_basics/generating_crd.html

	// The version of the installed release
	// +optional
	Version string `json:"version,omitempty"`
}

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KnativeServing is the Schema for the knativeservings API
// +k8s:openapi-gen=true
type KnativeServing struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KnativeServingSpec   `json:"spec,omitempty"`
	Status KnativeServingStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KnativeServingList contains a list of KnativeServing
type KnativeServingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KnativeServing `json:"items"`
}
