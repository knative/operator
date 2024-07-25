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

	"go.uber.org/zap"
	istiov1alpha3 "istio.io/api/networking/v1alpha3"
	istiov1beta1 "istio.io/api/networking/v1beta1"
	istionetworkingv1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	istionetworkingv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	"istio.io/client-go/pkg/clientset/versioned/scheme"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"knative.dev/operator/pkg/apis/operator/base"
	servingv1beta1 "knative.dev/operator/pkg/apis/operator/v1beta1"
	util "knative.dev/operator/pkg/reconciler/common/testing"
)

var log = zap.NewNop().Sugar()

func gatewayOverride(selector map[string]string, servers []*istiov1beta1.Server) *base.IstioGatewayOverride {
	return &base.IstioGatewayOverride{
		Selector: selector,
		Servers:  servers,
	}
}

func TestGatewayTransformV1alpha3(t *testing.T) {
	serverIn := []*istiov1alpha3.Server{
		{
			Hosts: []string{"localhost"},
			Port:  &istiov1alpha3.Port{Name: "test"},
		}, {
			Hosts: []string{"localhost"},
			Port:  &istiov1alpha3.Port{Name: "test"},
		}}

	serverUpdate := []*istiov1beta1.Server{
		{
			Hosts: []string{"localhost-1"},
			Port:  &istiov1beta1.Port{Name: "test-1", Protocol: "proto-1", Number: 25, TargetPort: 53},
		}, {
			Hosts: []string{"localhost-1"},
			Port:  &istiov1beta1.Port{Name: "test-1", Protocol: "proto-2", Number: 45, TargetPort: 23},
		}}

	tests := []struct {
		name                            string
		gatewayName                     string
		in                              map[string]string
		serversIn                       []*istiov1alpha3.Server
		knativeIngressGateway           *base.IstioGatewayOverride
		clusterLocalGateway             *base.IstioGatewayOverride
		deprecatedKnativeIngressGateway base.IstioGatewayOverride
		deprecatedClusterLocalGateway   base.IstioGatewayOverride
		expected                        map[string]string
		expectedServersIn               []*istiov1beta1.Server
	}{{
		name:        "update ingress gateway",
		gatewayName: "knative-ingress-gateway",
		in: map[string]string{
			"istio": "old-istio",
		},
		serversIn:             serverIn,
		knativeIngressGateway: gatewayOverride(map[string]string{"istio": "knative-ingress"}, serverUpdate),
		clusterLocalGateway:   gatewayOverride(map[string]string{"istio": "cluster-local"}, nil),
		expected: map[string]string{
			"istio": "knative-ingress",
		},
		expectedServersIn: serverUpdate,
	}, {
		name:        "update local gateway",
		gatewayName: "cluster-local-gateway",
		in: map[string]string{
			"istio": "old-istio",
		},
		knativeIngressGateway: gatewayOverride(map[string]string{"istio": "knative-ingress"}, nil),
		clusterLocalGateway:   gatewayOverride(map[string]string{"istio": "cluster-local"}, serverUpdate),
		expected: map[string]string{
			"istio": "cluster-local",
		},
		expectedServersIn: serverUpdate,
	}, {
		name:        "update ingress gateway with both new and deprecate config",
		gatewayName: "knative-ingress-gateway",
		in: map[string]string{
			"istio": "old-istio",
		},
		knativeIngressGateway:           gatewayOverride(map[string]string{"istio": "win"}, nil),
		deprecatedKnativeIngressGateway: *gatewayOverride(map[string]string{"istio": "lose"}, nil),
		expected: map[string]string{
			"istio": "win",
		},
	}, {
		name:        "update local gateway with both new and deprecate config",
		gatewayName: "cluster-local-gateway",
		in: map[string]string{
			"istio": "old-istio",
		},
		clusterLocalGateway:           gatewayOverride(map[string]string{"istio": "win"}, nil),
		deprecatedClusterLocalGateway: *gatewayOverride(map[string]string{"istio": "lose"}, nil),
		expected: map[string]string{
			"istio": "win",
		},
	}, {
		name:        "do not update unknown gateway",
		gatewayName: "not-knative-ingress-gateway",
		in: map[string]string{
			"istio": "old-istio",
		},
		knativeIngressGateway:           gatewayOverride(map[string]string{"istio": "knative-ingress"}, nil),
		clusterLocalGateway:             gatewayOverride(map[string]string{"istio": "cluster-local"}, nil),
		deprecatedKnativeIngressGateway: *gatewayOverride(map[string]string{"istio": "lose"}, nil),
		deprecatedClusterLocalGateway:   *gatewayOverride(map[string]string{"istio": "cluster-local"}, nil),
		expected: map[string]string{
			"istio": "old-istio",
		},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gateway := makeUnstructuredGatewayAlpha(tt.gatewayName, tt.in, tt.serversIn)
			instance := &servingv1beta1.KnativeServing{
				Spec: servingv1beta1.KnativeServingSpec{
					Ingress: &servingv1beta1.IngressConfigs{
						Istio: base.IstioIngressConfiguration{
							Enabled:               true,
							KnativeIngressGateway: tt.knativeIngressGateway,
							KnativeLocalGateway:   tt.clusterLocalGateway,
						},
					},
				},
			}

			gatewayTransform(instance, log)(gateway)

			gatewayResult := &istionetworkingv1alpha3.Gateway{}
			err := scheme.Scheme.Convert(gateway, gatewayResult, nil)
			util.AssertEqual(t, err, nil)
			util.AssertDeepEqual(t, gatewayResult.Spec.Selector, tt.expected)
			for i, server := range gatewayResult.Spec.Servers {
				util.AssertDeepEqual(t, server.Hosts, tt.expectedServersIn[i].Hosts)
				util.AssertDeepEqual(t, server.Port.Name, tt.expectedServersIn[i].Port.Name)
				util.AssertDeepEqual(t, server.Port.Number, tt.expectedServersIn[i].Port.Number)
				util.AssertDeepEqual(t, server.Port.Protocol, tt.expectedServersIn[i].Port.Protocol)
			}
		})
	}
}

func TestGatewayTransform(t *testing.T) {
	serverIn := []*istiov1beta1.Server{
		{
			Hosts: []string{"localhost"},
			Port:  &istiov1beta1.Port{Name: "test"},
		}, {
			Hosts: []string{"localhost"},
			Port:  &istiov1beta1.Port{Name: "test"},
		}}

	serverUpdate := []*istiov1beta1.Server{
		{
			Hosts: []string{"localhost-1"},
			Port:  &istiov1beta1.Port{Name: "test-1", Protocol: "proto-1", Number: 25, TargetPort: 53},
		}, {
			Hosts: []string{"localhost-1"},
			Port:  &istiov1beta1.Port{Name: "test-1", Protocol: "proto-2", Number: 45, TargetPort: 23},
		}}

	tests := []struct {
		name                            string
		gatewayName                     string
		in                              map[string]string
		serversIn                       []*istiov1beta1.Server
		knativeIngressGateway           *base.IstioGatewayOverride
		clusterLocalGateway             *base.IstioGatewayOverride
		deprecatedKnativeIngressGateway base.IstioGatewayOverride
		deprecatedClusterLocalGateway   base.IstioGatewayOverride
		expected                        map[string]string
		expectedServersIn               []*istiov1beta1.Server
	}{{
		name:        "update ingress gateway",
		gatewayName: "knative-ingress-gateway",
		in: map[string]string{
			"istio": "old-istio",
		},
		serversIn:             serverIn,
		knativeIngressGateway: gatewayOverride(map[string]string{"istio": "knative-ingress"}, serverUpdate),
		clusterLocalGateway:   gatewayOverride(map[string]string{"istio": "cluster-local"}, nil),
		expected: map[string]string{
			"istio": "knative-ingress",
		},
		expectedServersIn: serverUpdate,
	}, {
		name:        "update local gateway",
		gatewayName: "cluster-local-gateway",
		in: map[string]string{
			"istio": "old-istio",
		},
		knativeIngressGateway: gatewayOverride(map[string]string{"istio": "knative-ingress"}, nil),
		clusterLocalGateway:   gatewayOverride(map[string]string{"istio": "cluster-local"}, serverUpdate),
		expected: map[string]string{
			"istio": "cluster-local",
		},
		expectedServersIn: serverUpdate,
	}, {
		name:        "update ingress gateway with both new and deprecate config",
		gatewayName: "knative-ingress-gateway",
		in: map[string]string{
			"istio": "old-istio",
		},
		knativeIngressGateway:           gatewayOverride(map[string]string{"istio": "win"}, nil),
		deprecatedKnativeIngressGateway: *gatewayOverride(map[string]string{"istio": "lose"}, nil),
		expected: map[string]string{
			"istio": "win",
		},
	}, {
		name:        "update local gateway with both new and deprecate config",
		gatewayName: "cluster-local-gateway",
		in: map[string]string{
			"istio": "old-istio",
		},
		clusterLocalGateway:           gatewayOverride(map[string]string{"istio": "win"}, nil),
		deprecatedClusterLocalGateway: *gatewayOverride(map[string]string{"istio": "lose"}, nil),
		expected: map[string]string{
			"istio": "win",
		},
	}, {
		name:        "do not update unknown gateway",
		gatewayName: "not-knative-ingress-gateway",
		in: map[string]string{
			"istio": "old-istio",
		},
		knativeIngressGateway:           gatewayOverride(map[string]string{"istio": "knative-ingress"}, nil),
		clusterLocalGateway:             gatewayOverride(map[string]string{"istio": "cluster-local"}, nil),
		deprecatedKnativeIngressGateway: *gatewayOverride(map[string]string{"istio": "lose"}, nil),
		deprecatedClusterLocalGateway:   *gatewayOverride(map[string]string{"istio": "cluster-local"}, nil),
		expected: map[string]string{
			"istio": "old-istio",
		},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gateway := makeUnstructuredGateway(tt.gatewayName, tt.in, tt.serversIn)
			instance := &servingv1beta1.KnativeServing{
				Spec: servingv1beta1.KnativeServingSpec{
					Ingress: &servingv1beta1.IngressConfigs{
						Istio: base.IstioIngressConfiguration{
							Enabled:               true,
							KnativeIngressGateway: tt.knativeIngressGateway,
							KnativeLocalGateway:   tt.clusterLocalGateway,
						},
					},
				},
			}

			gatewayTransform(instance, log)(gateway)

			gatewayResult := &istionetworkingv1beta1.Gateway{}
			err := scheme.Scheme.Convert(gateway, gatewayResult, nil)
			util.AssertEqual(t, err, nil)
			util.AssertDeepEqual(t, gatewayResult.Spec.Selector, tt.expected)
			for i, server := range gatewayResult.Spec.Servers {
				util.AssertDeepEqual(t, server.Hosts, tt.expectedServersIn[i].Hosts)
				util.AssertDeepEqual(t, server.Port.Name, tt.expectedServersIn[i].Port.Name)
				util.AssertDeepEqual(t, server.Port.Number, tt.expectedServersIn[i].Port.Number)
				util.AssertDeepEqual(t, server.Port.Protocol, tt.expectedServersIn[i].Port.Protocol)
			}
		})
	}
}

func makeUnstructuredGateway(name string, selector map[string]string, servers []*istiov1beta1.Server) *unstructured.Unstructured {
	gateway := &istionetworkingv1beta1.Gateway{}
	result := &unstructured.Unstructured{}
	gateway.SetName(name)
	gateway.Spec.Selector = selector
	gateway.Spec.Servers = servers

	if err := scheme.Scheme.Convert(gateway, result, nil); err != nil {
		panic(err)
	}

	return result
}

func makeUnstructuredGatewayAlpha(name string, selector map[string]string, servers []*istiov1alpha3.Server) *unstructured.Unstructured {
	gateway := &istionetworkingv1alpha3.Gateway{}
	result := &unstructured.Unstructured{}
	gateway.SetName(name)
	gateway.Spec.Selector = selector
	gateway.Spec.Servers = servers

	if err := scheme.Scheme.Convert(gateway, result, nil); err != nil {
		panic(err)
	}

	return result
}
