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

type updateGatewayTest struct {
	name                  string
	gatewayName           string
	in                    map[string]string
	knativeIngressGateway servingv1alpha1.IstioGatewayOverride
	clusterLocalGateway   servingv1alpha1.IstioGatewayOverride
	expected              map[string]string
}

var updateGatewayTests = []updateGatewayTest{
	{
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
	},
	{
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
	},
	{
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
	},
}

func TestGatewayTransform(t *testing.T) {
	for _, tt := range updateGatewayTests {
		t.Run(tt.name, func(t *testing.T) {
			runGatewayTransformTest(t, &tt)
		})
	}
}
func runGatewayTransformTest(t *testing.T, tt *updateGatewayTest) {
	unstructedGateway := makeUnstructuredGateway(t, tt)
	instance := &servingv1alpha1.KnativeServing{
		Spec: servingv1alpha1.KnativeServingSpec{
			KnativeIngressGateway: tt.knativeIngressGateway,
			ClusterLocalGateway:   tt.clusterLocalGateway,
		},
	}
	gatewayTransform := GatewayTransform(instance, log)
	gatewayTransform(&unstructedGateway)
	validateUnstructedGatewayChanged(t, tt, &unstructedGateway)
}

func validateUnstructedGatewayChanged(t *testing.T, tt *updateGatewayTest, u *unstructured.Unstructured) {
	var gateway = &v1alpha3.Gateway{}
	err := scheme.Scheme.Convert(u, gateway, nil)
	util.AssertEqual(t, err, nil)
	for expectedKey, expectedValue := range tt.expected {
		util.AssertEqual(t, gateway.Spec.Selector[expectedKey], expectedValue)
	}
}

func makeUnstructuredGateway(t *testing.T, tt *updateGatewayTest) unstructured.Unstructured {
	gateway := v1alpha3.Gateway{
		Spec: istiov1alpha3.Gateway{
			Selector: tt.in,
		},
	}
	gateway.APIVersion = "networking.istio.io/v1alpha3"
	gateway.Kind = "Gateway"
	gateway.Name = tt.gatewayName
	result := unstructured.Unstructured{}
	err := scheme.Scheme.Convert(&gateway, &result, nil)
	if err != nil {
		t.Fatalf("Could not create unstructured deployment object: %v, err: %v", result, err)
	}
	return result
}
