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

package base

import (
	istiov1alpha3 "istio.io/api/networking/v1alpha3"
	v1 "k8s.io/api/core/v1"
)

// IstioIngressConfiguration specifies options for the istio ingresses.
type IstioIngressConfiguration struct {
	Enabled bool `json:"enabled"`

	// KnativeIngressGateway overrides the knative-ingress-gateway.
	// +optional
	KnativeIngressGateway *IstioGatewayOverride `json:"knative-ingress-gateway,omitempty"`

	// KnativeLocalGateway overrides the knative-local-gateway.
	// +optional
	KnativeLocalGateway *IstioGatewayOverride `json:"knative-local-gateway,omitempty"`
}

// KourierIngressConfiguration specifies whether to enable the kourier ingresses.
type KourierIngressConfiguration struct {
	Enabled bool `json:"enabled"`

	// ServiceType specifies the service type for kourier gateway.
	ServiceType v1.ServiceType `json:"service-type,omitempty"`
}

// ContourIngressConfiguration specifies whether to enable the contour ingresses.
type ContourIngressConfiguration struct {
	Enabled bool `json:"enabled"`
}

// IstioGatewayOverride override the knative-ingress-gateway and knative-local-gateway(cluster-local-gateway)
type IstioGatewayOverride struct {
	// A map of values to replace the "selector" values in the knative-ingress-gateway and knative-local-gateway(cluster-local-gateway)
	Selector map[string]string `json:"selector,omitempty"`

	// A list of server specifications.
	Servers []*istiov1alpha3.Server `json:"servers,omitempty"`
}
