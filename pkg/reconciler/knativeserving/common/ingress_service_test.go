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

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	util "knative.dev/operator/pkg/reconciler/common/testing"
)

func TestIngressServiceTransform(t *testing.T) {
	tests := []struct {
		name              string
		namespace         string
		serviceName       string
		expectedNamespace string
		expected          bool
	}{{
		name:              "KeepKnativeIngressServiceNamespace",
		namespace:         "test-namespace",
		serviceName:       "knative-local-gateway",
		expectedNamespace: "istio-system",
		expected:          true,
	}, {
		name:              "DoNotKeepKnativeIngressServiceNamespace",
		namespace:         "test-namespace",
		serviceName:       "knative-local-gateway-other",
		expectedNamespace: "istio-system",
		expected:          false,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := makeIngressService(t, tt.serviceName, tt.namespace)
			IngressServiceTransform()(service)
			util.AssertEqual(t, service.GetNamespace() == tt.expectedNamespace, tt.expected)
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
