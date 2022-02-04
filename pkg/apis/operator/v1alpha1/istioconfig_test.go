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
	"testing"

	"knative.dev/operator/pkg/apis/operator/base"
	util "knative.dev/operator/pkg/reconciler/common/testing"
)

func TestConvertToIstioConfig(t *testing.T) {
	for _, tt := range []struct {
		name                string
		ks                  *KnativeServing
		expectedIstioConfig base.IstioIngressConfiguration
	}{{
		name: "Deprecated Ingress Gateway will be passed into the istio configuration",
		ks: &KnativeServing{
			Spec: KnativeServingSpec{
				DeprecatedKnativeIngressGateway: base.IstioGatewayOverride{
					Selector: map[string]string{"istio": "knative-ingress"},
				},
				DeprecatedClusterLocalGateway: base.IstioGatewayOverride{
					Selector: map[string]string{"istio": "cluster-local"},
				},
			},
		},
		expectedIstioConfig: base.IstioIngressConfiguration{
			KnativeIngressGateway: &base.IstioGatewayOverride{
				Selector: map[string]string{"istio": "knative-ingress"},
			},
			KnativeLocalGateway: &base.IstioGatewayOverride{
				Selector: map[string]string{"istio": "cluster-local"},
			},
		},
	}, {
		name: "Ingress Gateway will be passed into the istio configuration",
		ks: &KnativeServing{
			Spec: KnativeServingSpec{
				DeprecatedKnativeIngressGateway: base.IstioGatewayOverride{
					Selector: map[string]string{"istio": "knative-ingress"},
				},
				DeprecatedClusterLocalGateway: base.IstioGatewayOverride{
					Selector: map[string]string{"istio": "cluster-local"},
				},
				Ingress: &IngressConfigs{
					Istio: base.IstioIngressConfiguration{
						KnativeIngressGateway: &base.IstioGatewayOverride{
							Selector: map[string]string{"istio": "knative-ingress-istio"},
						},
						KnativeLocalGateway: &base.IstioGatewayOverride{
							Selector: map[string]string{"istio": "cluster-local-istio"},
						},
					},
				},
			},
		},
		expectedIstioConfig: base.IstioIngressConfiguration{
			KnativeIngressGateway: &base.IstioGatewayOverride{
				Selector: map[string]string{"istio": "knative-ingress-istio"},
			},
			KnativeLocalGateway: &base.IstioGatewayOverride{
				Selector: map[string]string{"istio": "cluster-local-istio"},
			},
		},
	}, {
		name: "Deprecated ingress Gateway will be passed into the istio configuration if Ingress Gateway is not available",
		ks: &KnativeServing{
			Spec: KnativeServingSpec{
				DeprecatedKnativeIngressGateway: base.IstioGatewayOverride{
					Selector: map[string]string{"istio": "knative-ingress"},
				},
				DeprecatedClusterLocalGateway: base.IstioGatewayOverride{
					Selector: map[string]string{"istio": "cluster-local"},
				},
				Ingress: &IngressConfigs{
					Istio: base.IstioIngressConfiguration{
						KnativeIngressGateway: &base.IstioGatewayOverride{
							Selector: map[string]string{"istio": "knative-ingress-istio"},
						},
					},
				},
			},
		},
		expectedIstioConfig: base.IstioIngressConfiguration{
			KnativeIngressGateway: &base.IstioGatewayOverride{
				Selector: map[string]string{"istio": "knative-ingress-istio"},
			},
			KnativeLocalGateway: &base.IstioGatewayOverride{
				Selector: map[string]string{"istio": "cluster-local"},
			},
		},
	}} {
		t.Run(tt.name, func(t *testing.T) {
			istioConfig := ConvertToIstioConfig(tt.ks)
			util.AssertDeepEqual(t, istioConfig, tt.expectedIstioConfig)
		})
	}
}
