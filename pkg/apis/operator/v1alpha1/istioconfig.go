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

package v1alpha1

import (
	"knative.dev/operator/pkg/apis/operator/base"
)

// ConvertToIstioConfig merges the gateway config into the ingress istio config
func ConvertToIstioConfig(source *KnativeServing) base.IstioIngressConfiguration {
	istioConfig := base.IstioIngressConfiguration{}
	if source.Spec.Ingress != nil {
		istioConfig = source.Spec.Ingress.Istio
	}

	if istioConfig.KnativeIngressGateway == nil && source.Spec.DeprecatedKnativeIngressGateway.Selector != nil {
		istioConfig.KnativeIngressGateway = &source.Spec.DeprecatedKnativeIngressGateway
	}
	if istioConfig.KnativeLocalGateway == nil && source.Spec.DeprecatedClusterLocalGateway.Selector != nil {
		istioConfig.KnativeLocalGateway = &source.Spec.DeprecatedClusterLocalGateway
	}

	return istioConfig
}
