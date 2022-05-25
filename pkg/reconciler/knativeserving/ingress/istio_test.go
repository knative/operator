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
	"strconv"
	"testing"

	mf "github.com/manifestival/manifestival"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	istiov1alpha3 "istio.io/api/networking/v1alpha3"
	istionetworkingv1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	"istio.io/client-go/pkg/clientset/versioned/scheme"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"knative.dev/operator/pkg/apis/operator/base"
	servingv1beta1 "knative.dev/operator/pkg/apis/operator/v1beta1"
	util "knative.dev/operator/pkg/reconciler/common/testing"
)

var log = zap.NewNop().Sugar()

func gatewayOverride(selector map[string]string, servers []*istiov1alpha3.Server) *base.IstioGatewayOverride {
	return &base.IstioGatewayOverride{
		Selector: selector,
		Servers:  servers,
	}
}

func TestGatewayTransform(t *testing.T) {
	serverIn := []*istiov1alpha3.Server{
		{
			Hosts: []string{"localhost"},
			Port:  &istiov1alpha3.Port{Name: "test"},
		}, {
			Hosts: []string{"localhost"},
			Port:  &istiov1alpha3.Port{Name: "test"},
		}}

	serverUpdate := []*istiov1alpha3.Server{
		{
			Hosts: []string{"localhost-1"},
			Port:  &istiov1alpha3.Port{Name: "test-1", Protocol: "proto-1", Number: 25, TargetPort: 53},
		}, {
			Hosts: []string{"localhost-1"},
			Port:  &istiov1alpha3.Port{Name: "test-1", Protocol: "proto-2", Number: 45, TargetPort: 23},
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
		expectedServersIn               []*istiov1alpha3.Server
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
			gateway := makeUnstructuredGateway(t, tt.gatewayName, tt.in, tt.serversIn)
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
				util.AssertDeepEqual(t, server.Port.TargetPort, tt.expectedServersIn[i].Port.TargetPort)
				util.AssertDeepEqual(t, server.Port.Protocol, tt.expectedServersIn[i].Port.Protocol)
			}
		})
	}
}

func TestInformerFiltering(t *testing.T) {
	logger := zaptest.NewLogger(t)
	tests := []struct {
		name     string
		instance servingv1beta1.KnativeServing
		enable   bool
	}{{
		name: "enable secret informer filtering",
		instance: servingv1beta1.KnativeServing{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{enableSecretInformerFilteringByCertUIDAnno: "true"},
			},
		},
		enable: true,
	}, {
		name: "disable secret informer filtering",
		instance: servingv1beta1.KnativeServing{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{enableSecretInformerFilteringByCertUIDAnno: "false"},
			},
		},
		enable: false,
	}, {
		name:     "do not configure secret informer filtering",
		instance: servingv1beta1.KnativeServing{},
		enable:   false,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, _ := mf.ManifestFrom(mf.Slice{})
			depM, err := createNetIstioDeploymentManifest()
			if err != nil {
				t.Fatalf("Could not create unstructured net-istio deployment, err: %v", err)
			}
			m = m.Append(*depM)
			util.AssertEqual(t, len(m.Resources()), 1)
			if tt.enable {
				transformer := enableSecretInformerFilteringByCertUID(&tt.instance, logger.Sugar())
				m, err = m.Transform(transformer)
				if err != nil {
					t.Fatalf("Could not transform the net-istio deployment, err: %v", err)
				}
			}
			got := &appsv1.Deployment{}
			if err := scheme.Scheme.Convert(&m.Resources()[0], got, nil); err != nil {
				t.Fatalf("Unable to convert Unstructured to Deployment: %s", err)
			}
			if tt.enable {
				util.AssertEqual(t, got.Spec.Template.Spec.Containers[0].Env[0].Name, "ENABLE_SECRET_INFORMER_FILTERING_BY_CERT_UID")
				util.AssertEqual(t, got.Spec.Template.Spec.Containers[0].Env[0].Value, strconv.FormatBool(tt.enable))
			} else {
				util.AssertEqual(t, len(got.Spec.Template.Spec.Containers[0].Env), 0)

			}
		})
	}
}

func createNetIstioDeploymentManifest() (*mf.Manifest, error) {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "net-istio-controller",
			Namespace: "knative-serving",
		},
		Spec: appsv1.DeploymentSpec{
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name: "controller",
						},
					},
				},
			}},
	}
	var depU = &unstructured.Unstructured{}
	if err := scheme.Scheme.Convert(deployment, depU, nil); err != nil {
		return nil, err
	}
	manifest, err := mf.ManifestFrom(mf.Slice([]unstructured.Unstructured{*depU}))
	if err != nil {
		return nil, err
	}
	return &manifest, nil
}

func makeUnstructuredGateway(t *testing.T, name string, selector map[string]string, servers []*istiov1alpha3.Server) *unstructured.Unstructured {
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
