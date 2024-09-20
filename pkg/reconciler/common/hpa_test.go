/*
Copyright 2023 The Knative Authors

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

	v2 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
	util "knative.dev/operator/pkg/reconciler/common/testing"
)

func TestHpaTransform(t *testing.T) {
	cases := []struct {
		name     string
		in       *unstructured.Unstructured
		replicas int64
		expected *unstructured.Unstructured
		err      error
	}{{
		name:     "Object is not a HPA",
		in:       makeUnstructuredDeployment(t, "not-a-hpa"),
		replicas: 5,
		expected: makeUnstructuredDeployment(t, "not-a-hpa"),
		err:      nil,
	}, {
		name:     "Kafka Dispatcher is custom autoscaler",
		in:       makeUnstructuredDeployment(t, "kafka-source-dispatcher"),
		replicas: 5,
		expected: makeUnstructuredDeploymentReplicas(t, "kafka-source-dispatcher", 1),
		err:      nil,
	}, {
		name:     "minReplicas same as override",
		in:       makeUnstructuredHPA(t, "hpa", 1, 2),
		replicas: 1,
		expected: makeUnstructuredHPA(t, "hpa", 1, 2),
		err:      nil,
	}, {
		name:     "minReplicas lower than override",
		in:       makeUnstructuredHPA(t, "hpa", 1, 2),
		replicas: 5,
		expected: makeUnstructuredHPA(t, "hpa", 5, 6),
		err:      nil,
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {

			err := hpaTransform(tc.in, tc.replicas)

			util.AssertDeepEqual(t, err, tc.err)
			util.AssertDeepEqual(t, tc.in, tc.expected)
		})
	}
}

func TestGetHPAName(t *testing.T) {
	util.AssertEqual(t, getHPAName("mt-broker-ingress"), "broker-ingress-hpa")
	util.AssertEqual(t, getHPAName("activator"), "activator")
}

func makeUnstructuredHPA(t *testing.T, name string, minReplicas, maxReplicas int32) *unstructured.Unstructured {
	hpa := &v2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v2.HorizontalPodAutoscalerSpec{
			MinReplicas: &minReplicas,
			MaxReplicas: maxReplicas,
		},
	}

	result := &unstructured.Unstructured{}
	err := scheme.Scheme.Convert(hpa, result, nil)
	if err != nil {
		t.Fatalf("Could not create unstructured HPA: %v, err: %v", hpa, err)
	}

	return result
}
