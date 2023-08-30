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

package common

import (
	"testing"

	"knative.dev/operator/pkg/apis/operator/base"
	"knative.dev/operator/pkg/apis/operator/v1beta1"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"

	util "knative.dev/operator/pkg/reconciler/common/testing"
)

func TestHighAvailabilityTransform(t *testing.T) {
	cases := []struct {
		name     string
		config   *base.HighAvailability
		in       *unstructured.Unstructured
		expected *unstructured.Unstructured
		err      error
	}{{
		name:     "HA; controller",
		config:   makeHa(2),
		in:       makeUnstructuredDeployment(t, "controller"),
		expected: makeUnstructuredDeploymentReplicas(t, "controller", 2),
	}, {
		name:     "HA; autoscaler",
		config:   makeHa(2),
		in:       makeUnstructuredDeployment(t, "autoscaler"),
		expected: makeUnstructuredDeploymentReplicas(t, "autoscaler", 2),
	}, {
		name:     "HA; unsupported deployment",
		config:   makeHa(2),
		in:       makeUnstructuredDeployment(t, "pingsource-mt-adapter"),
		expected: makeUnstructuredDeployment(t, "pingsource-mt-adapter"),
	}, {
		name:     "HA; adjust hpa",
		config:   makeHa(2),
		in:       makeUnstructuredHPA(t, "activator", 1, 4),
		expected: makeUnstructuredHPA(t, "activator", 2, 5),
	}, {
		name:     "HA; do not adjust deployment directory which has HPA", // The replica should be djusted by HPA.
		config:   makeHa(2),
		in:       makeUnstructuredDeployment(t, "activator"),
		expected: makeUnstructuredDeployment(t, "activator"),
	}, {
		name:     "HA; keep higher hpa value",
		config:   makeHa(2),
		in:       makeUnstructuredHPA(t, "activator", 3, 5),
		expected: makeUnstructuredHPA(t, "activator", 3, 5),
	}, {
		name:     "HA; do nothing when replicas is equal to minReplicas",
		config:   makeHa(2),
		in:       makeUnstructuredHPA(t, "activator", 2, 5),
		expected: makeUnstructuredHPA(t, "activator", 2, 5),
	}, {
		name:     "HA; adjust hpa when replicas is larger than maxReplicas",
		config:   makeHa(6),
		in:       makeUnstructuredHPA(t, "activator", 2, 5),
		expected: makeUnstructuredHPA(t, "activator", 6, 9), // maxReplicas is increased by max+(replicas-min) to avoid minReplicas > maxReplicas happenning.
	}, {
		name:     "HA; adjust hpa when minReplica is equal to maxReplicas",
		config:   makeHa(3),
		in:       makeUnstructuredHPA(t, "activator", 2, 2),
		expected: makeUnstructuredHPA(t, "activator", 3, 3),
	}, {
		name: "HA; empty replicas",
		config: &base.HighAvailability{
			Replicas: nil,
		},
		in:       makeUnstructuredHPA(t, "activator", 2, 2),
		expected: makeUnstructuredHPA(t, "activator", 2, 2),
	}, {
		name:     "HA; empty high availability",
		config:   nil,
		in:       makeUnstructuredHPA(t, "activator", 2, 2),
		expected: makeUnstructuredHPA(t, "activator", 2, 2),
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {

			instance := &v1beta1.KnativeServing{
				Spec: v1beta1.KnativeServingSpec{
					CommonSpec: base.CommonSpec{
						HighAvailability: tc.config,
					},
				},
			}
			haTransform := HighAvailabilityTransform(instance)
			err := haTransform(tc.in)

			util.AssertDeepEqual(t, err, tc.err)
			util.AssertDeepEqual(t, tc.in, tc.expected)
		})
	}
}

func makeHa(replicas int32) *base.HighAvailability {
	return &base.HighAvailability{
		Replicas: &replicas,
	}
}

func makeUnstructuredDeployment(t *testing.T, name string) *unstructured.Unstructured {
	return makeUnstructuredDeploymentReplicas(t, name, 1)
}

func makeUnstructuredDeploymentReplicas(t *testing.T, name string, replicas int32) *unstructured.Unstructured {
	d := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
		},
	}
	result := &unstructured.Unstructured{}
	err := scheme.Scheme.Convert(d, result, nil)
	if err != nil {
		t.Fatalf("Could not create unstructured Deployment: %v, err: %v", d, err)
	}

	return result
}
