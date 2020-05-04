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
package common

import (
	mf "github.com/manifestival/manifestival"
	"knative.dev/operator/pkg/reconciler/common"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"

	istiov1alpha3 "istio.io/api/networking/v1alpha3"
	"istio.io/client-go/pkg/apis/networking/v1alpha3"
	servingv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
	util "knative.dev/operator/pkg/reconciler/common/testing"
)

func init() {
	v1alpha3.AddToScheme(scheme.Scheme)
}

func TestGatewayTransform(t *testing.T) {
	tests := []struct {
		name                  string
		gatewayName           string
		in                    map[string]string
		knativeIngressGateway servingv1alpha1.IstioGatewayOverride
		clusterLocalGateway   servingv1alpha1.IstioGatewayOverride
		expected              map[string]string
		expectedPolicy        mf.Predicate
	}{{
		name:        "UpdatesKnativeIngressGateway",
		gatewayName: "knative-ingress-gateway",
		in: map[string]string{
			"istio": "old-istio",
		},
		knativeIngressGateway: servingv1alpha1.IstioGatewayOverride{
			Selector: map[string]string{
				"istio": "knative-ingress",
			},
		},
		clusterLocalGateway: servingv1alpha1.IstioGatewayOverride{
			Selector: map[string]string{
				"istio": "cluster-local",
			},
		},
		expected: map[string]string{
			"istio": "knative-ingress",
		},
	}, {
		name:        "UpdatesClusterLocalGateway",
		gatewayName: "cluster-local-gateway",
		in: map[string]string{
			"istio": "old-istio",
		},
		knativeIngressGateway: servingv1alpha1.IstioGatewayOverride{
			Selector: map[string]string{
				"istio": "knative-ingress",
			},
		},
		clusterLocalGateway: servingv1alpha1.IstioGatewayOverride{
			Selector: map[string]string{
				"istio": "cluster-local",
			},
		},
		expected: map[string]string{
			"istio": "cluster-local",
		},
	}, {
		name:        "DoesNothingToOtherGateway",
		gatewayName: "not-knative-ingress-gateway",
		in: map[string]string{
			"istio": "old-istio",
		},
		knativeIngressGateway: servingv1alpha1.IstioGatewayOverride{
			Selector: map[string]string{
				"istio": "knative-ingress",
			},
		},
		clusterLocalGateway: servingv1alpha1.IstioGatewayOverride{
			Selector: map[string]string{
				"istio": "cluster-local",
			},
		},
		expected: map[string]string{
			"istio": "old-istio",
		},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unstructedGateway := makeUnstructuredGateway(t, tt.gatewayName, tt.in)
			instance := &servingv1alpha1.KnativeServing{
				Spec: servingv1alpha1.KnativeServingSpec{
					KnativeIngressGateway: tt.knativeIngressGateway,
					ClusterLocalGateway:   tt.clusterLocalGateway,
				},
			}
			manifestWithPolicy := &common.ManifestWithPolicy {
				GlobalPredicate: mf.All(),
			}
			gatewayTransform := GatewayTransform(instance, log, manifestWithPolicy)
			gatewayTransform(&unstructedGateway)

			var gateway = &v1alpha3.Gateway{}
			err := scheme.Scheme.Convert(&unstructedGateway, gateway, nil)
			util.AssertEqual(t, err, nil)
			for expectedKey, expectedValue := range tt.expected {
				util.AssertEqual(t, gateway.Spec.Selector[expectedKey], expectedValue)
			}
		})
	}
}

func makeUnstructuredGateway(t *testing.T, name string, selector map[string]string) unstructured.Unstructured {
	gateway := v1alpha3.Gateway{
		Spec: istiov1alpha3.Gateway{
			Selector: selector,
		},
	}
	gateway.APIVersion = "networking.istio.io/v1alpha3"
	gateway.Kind = "Gateway"
	gateway.Name = name
	result := unstructured.Unstructured{}
	err := scheme.Scheme.Convert(&gateway, &result, nil)
	if err != nil {
		t.Fatalf("Could not create unstructured deployment object: %v, err: %v", result, err)
	}
	return result
}
