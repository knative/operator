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

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
	"knative.dev/operator/pkg/apis/operator/base"
	servingv1beta1 "knative.dev/operator/pkg/apis/operator/v1beta1"
	util "knative.dev/operator/pkg/reconciler/common/testing"
)

func TestIngressServiceTransform(t *testing.T) {
	tests := []struct {
		name              string
		namespace         string
		serviceName       string
		expectedNamespace string
		instance          *servingv1beta1.KnativeServing
		expected          bool
	}{{
		name:              "KeepKnativeIngressServiceNamespace",
		namespace:         "test-namespace",
		serviceName:       "knative-local-gateway",
		expectedNamespace: "istio-system",
		instance: &servingv1beta1.KnativeServing{
			Spec: servingv1beta1.KnativeServingSpec{},
		},
		expected: true,
	}, {
		name:              "DoNotKeepKnativeIngressServiceNamespace",
		namespace:         "test-namespace",
		serviceName:       "knative-local-gateway-other",
		expectedNamespace: "istio-system",
		instance: &servingv1beta1.KnativeServing{
			Spec: servingv1beta1.KnativeServingSpec{},
		},
		expected: false,
	}, {
		name:              "IstioNotUnderDefaultNS with istio",
		namespace:         "test-namespace",
		serviceName:       "knative-local-gateway",
		expectedNamespace: "istio-system-1",
		instance: &servingv1beta1.KnativeServing{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-instance",
				Namespace: "test-namespace",
			},
			Spec: servingv1beta1.KnativeServingSpec{
				CommonSpec: base.CommonSpec{
					Config: map[string]map[string]string{"istio": {"local-gateway.test-namespace.knative-local-gateway": "knative-local-gateway.istio-system-1.svc.cluster.local"}},
				},
			},
		},
		expected: true,
	}, {
		name:              "IstioNotUnderDefaultNS with config-istio",
		namespace:         "test-namespace",
		serviceName:       "knative-local-gateway",
		expectedNamespace: "istio-system-1",
		instance: &servingv1beta1.KnativeServing{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-instance",
				Namespace: "test-namespace",
			},
			Spec: servingv1beta1.KnativeServingSpec{
				CommonSpec: base.CommonSpec{
					Config: map[string]map[string]string{"config-istio": {"local-gateway.test-namespace.knative-local-gateway": "knative-local-gateway.istio-system-1.svc.cluster.local"}},
				},
			},
		},
		expected: true,
	}, {
		name:              "IstioNotUnderDefaultNS with both istio and config-istio",
		namespace:         "test-namespace",
		serviceName:       "knative-local-gateway",
		expectedNamespace: "istio-system-3",
		instance: &servingv1beta1.KnativeServing{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-instance",
				Namespace: "test-namespace",
			},
			Spec: servingv1beta1.KnativeServingSpec{
				CommonSpec: base.CommonSpec{
					Config: map[string]map[string]string{"istio": {"local-gateway.test-namespace.knative-local-gateway": "knative-local-gateway.istio-system-2.svc.cluster.local"},
						"config-istio": {"local-gateway.test-namespace.knative-local-gateway": "knative-local-gateway.istio-system-3.svc.cluster.local"}},
				},
			},
		},
		expected: true,
	}, {
		name:              "IstioNotUnderDefaultNS with invalid config-istio data",
		namespace:         "test-namespace",
		serviceName:       "knative-local-gateway",
		expectedNamespace: "istio-system",
		instance: &servingv1beta1.KnativeServing{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-instance",
				Namespace: "test-namespace",
			},
			Spec: servingv1beta1.KnativeServingSpec{
				CommonSpec: base.CommonSpec{
					Config: map[string]map[string]string{"config-istio": {"local-gateway.test-namespace.knative-local-gateway": "knative-local-gateway"}},
				},
			},
		},
		expected: true,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := makeIngressService(t, tt.serviceName, tt.namespace)
			IngressServiceTransform(tt.instance)(service)
			util.AssertEqual(t, service.GetNamespace() == tt.expectedNamespace, tt.expected)
			util.AssertEqual(t, service.GetOwnerReferences() == nil, true)
		})
	}
}

func makeIngressService(t *testing.T, name, ns string) *unstructured.Unstructured {
	service := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
	}
	result := &unstructured.Unstructured{}
	err := scheme.Scheme.Convert(service, result, nil)
	if err != nil {
		t.Fatalf("Could not create unstructured Service: %v, err: %v", service, err)
	}

	return result
}
