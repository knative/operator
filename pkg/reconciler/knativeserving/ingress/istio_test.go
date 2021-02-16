/*
Copyright 2020 The Knative Authors

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

package ingress

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	servingv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
	util "knative.dev/operator/pkg/reconciler/common/testing"
)

var log = zap.NewNop().Sugar()

func gatewayOverride(selector map[string]string) *servingv1alpha1.IstioGatewayOverride {
	return &servingv1alpha1.IstioGatewayOverride{
		Selector: selector,
	}
}

func TestGatewayTransform(t *testing.T) {
	tests := []struct {
		name                            string
		gatewayName                     string
		in                              map[string]string
		knativeIngressGateway           *servingv1alpha1.IstioGatewayOverride
		clusterLocalGateway             *servingv1alpha1.IstioGatewayOverride
		deprecatedKnativeIngressGateway servingv1alpha1.IstioGatewayOverride
		deprecatedClusterLocalGateway   servingv1alpha1.IstioGatewayOverride
		expected                        map[string]string
	}{{
		name:        "update ingress gateway",
		gatewayName: "knative-ingress-gateway",
		in: map[string]string{
			"istio": "old-istio",
		},
		knativeIngressGateway: gatewayOverride(map[string]string{"istio": "knative-ingress"}),
		clusterLocalGateway:   gatewayOverride(map[string]string{"istio": "cluster-local"}),
		expected: map[string]string{
			"istio": "knative-ingress",
		},
	}, {
		name:        "update local gateway",
		gatewayName: "cluster-local-gateway",
		in: map[string]string{
			"istio": "old-istio",
		},
		knativeIngressGateway: gatewayOverride(map[string]string{"istio": "knative-ingress"}),
		clusterLocalGateway:   gatewayOverride(map[string]string{"istio": "cluster-local"}),
		expected: map[string]string{
			"istio": "cluster-local",
		},
	}, {
		name:        "update ingress gateway with deprecatd setting",
		gatewayName: "knative-ingress-gateway",
		in: map[string]string{
			"istio": "old-istio",
		},
		deprecatedKnativeIngressGateway: *gatewayOverride(map[string]string{"istio": "knative-ingress"}),
		deprecatedClusterLocalGateway:   *gatewayOverride(map[string]string{"istio": "cluster-local"}),
		expected: map[string]string{
			"istio": "knative-ingress",
		},
	}, {
		name:        "update local gateway with deprecatd setting",
		gatewayName: "cluster-local-gateway",
		in: map[string]string{
			"istio": "old-istio",
		},
		deprecatedKnativeIngressGateway: *gatewayOverride(map[string]string{"istio": "knative-ingress"}),
		deprecatedClusterLocalGateway:   *gatewayOverride(map[string]string{"istio": "cluster-local"}),
		expected: map[string]string{
			"istio": "cluster-local",
		},
	}, {
		name:        "update ingress gateway with both new and deprecate config",
		gatewayName: "knative-ingress-gateway",
		in: map[string]string{
			"istio": "old-istio",
		},
		knativeIngressGateway:           gatewayOverride(map[string]string{"istio": "win"}),
		deprecatedKnativeIngressGateway: *gatewayOverride(map[string]string{"istio": "lose"}),
		expected: map[string]string{
			"istio": "win",
		},
	}, {
		name:        "update local gateway with both new and deprecate config",
		gatewayName: "cluster-local-gateway",
		in: map[string]string{
			"istio": "old-istio",
		},
		clusterLocalGateway:           gatewayOverride(map[string]string{"istio": "win"}),
		deprecatedClusterLocalGateway: *gatewayOverride(map[string]string{"istio": "lose"}),
		expected: map[string]string{
			"istio": "win",
		},
	}, {
		name:        "do not update unknown gateway",
		gatewayName: "not-knative-ingress-gateway",
		in: map[string]string{
			"istio": "old-istio",
		},
		knativeIngressGateway:           gatewayOverride(map[string]string{"istio": "knative-ingress"}),
		clusterLocalGateway:             gatewayOverride(map[string]string{"istio": "cluster-local"}),
		deprecatedKnativeIngressGateway: *gatewayOverride(map[string]string{"istio": "lose"}),
		deprecatedClusterLocalGateway:   *gatewayOverride(map[string]string{"istio": "cluster-local"}),
		expected: map[string]string{
			"istio": "old-istio",
		},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gateway := makeUnstructuredGateway(t, tt.gatewayName, tt.in)
			instance := &servingv1alpha1.KnativeServing{
				Spec: servingv1alpha1.KnativeServingSpec{
					Ingress: &servingv1alpha1.IngressConfigs{
						Istio: servingv1alpha1.IstioIngressConfiguration{
							Enabled:               true,
							KnativeIngressGateway: tt.knativeIngressGateway,
							KnativeLocalGateway:   tt.clusterLocalGateway,
						},
					},
					DeprecatedKnativeIngressGateway: tt.deprecatedKnativeIngressGateway,
					DeprecatedClusterLocalGateway:   tt.deprecatedClusterLocalGateway,
				},
			}

			gatewayTransform(instance, log)(gateway)

			got, ok, err := unstructured.NestedStringMap(gateway.Object, "spec", "selector")
			util.AssertEqual(t, err, nil)
			util.AssertEqual(t, ok, true)

			if !cmp.Equal(got, tt.expected) {
				t.Errorf("Got = %v, want: %v, diff:\n%s", got, tt.expected, cmp.Diff(got, tt.expected))
			}
		})
	}
}

func makeUnstructuredGateway(t *testing.T, name string, selector map[string]string) *unstructured.Unstructured {
	result := &unstructured.Unstructured{}
	result.SetAPIVersion("networking.istio.io/v1alpha3")
	result.SetKind("Gateway")
	result.SetName(name)
	unstructured.SetNestedStringMap(result.Object, selector, "spec", "selector")

	return result
}
