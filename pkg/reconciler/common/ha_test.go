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

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2beta1 "k8s.io/api/autoscaling/v2beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"

	operatorv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
	util "knative.dev/operator/pkg/reconciler/common/testing"
)

func TestHighAvailabilityTransform(t *testing.T) {
	cases := []struct {
		name     string
		config   *operatorv1alpha1.HighAvailability
		in       *unstructured.Unstructured
		expected *unstructured.Unstructured
		err      error
	}{{
		name:     "No HA; ConfigMap",
		config:   nil,
		in:       makeUnstructuredConfigMap(t, nil),
		expected: makeUnstructuredConfigMap(t, nil),
	}, {
		name:   "HA; ConfigMap",
		config: makeHa(2, "foo,bar"),
		in:     makeUnstructuredConfigMap(t, nil),
		expected: makeUnstructuredConfigMap(t, map[string]string{
			enabledComponentsKey: "foo,bar",
		}),
	}, {
		name:     "HA; HA deployment",
		config:   makeHa(2, "foo,bar"),
		in:       makeUnstructuredDeployment(t, "controller", map[string]string{"knative.dev/high-availability": "true"}),
		expected: makeUnstructuredDeploymentReplicasWithLabels(t, "controller", 2, map[string]string{"knative.dev/high-availability": "true"}),
	}, {
		name:     "HA; no HA deployment - no HA info",
		config:   makeHa(2, "foo,bar"),
		in:       makeUnstructuredDeployment(t, "some-unsupported-controller", map[string]string{}),
		expected: makeUnstructuredDeploymentReplicas(t, "some-unsupported-controller", 1),
	}, {
		name:     "HA; no HA deployment - HA false",
		config:   makeHa(2, "foo,bar"),
		in:       makeUnstructuredDeployment(t, "some-unsupported-controller", map[string]string{"knative.dev/high-availability": "false"}),
		expected: makeUnstructuredDeploymentReplicasWithLabels(t, "some-unsupported-controller", 1, map[string]string{"knative.dev/high-availability": "false"}),
	}, {
		name:     "HA; adjust hpa",
		config:   makeHa(2, "foo,bar"),
		in:       makeUnstructuredHPA(t, "activator", 1),
		expected: makeUnstructuredHPA(t, "activator", 2),
	}, {
		name:     "HA; keep higher hpa value",
		config:   makeHa(2, "foo,bar"),
		in:       makeUnstructuredHPA(t, "activator", 3),
		expected: makeUnstructuredHPA(t, "activator", 3),
	}}

	for i := range cases {
		tc := cases[i]

		haTransform := HighAvailabilityTransform(tc.config, log)
		err := haTransform(tc.in)

		util.AssertDeepEqual(t, err, tc.err)
		util.AssertDeepEqualWithName(t, tc.name, tc.in, tc.expected)
	}
}

func makeHa(replicas int32, leaderElectedComponents string) *operatorv1alpha1.HighAvailability {
	return &operatorv1alpha1.HighAvailability{
		Replicas:                replicas,
		LeaderElectedComponents: leaderElectedComponents,
	}
}

func makeUnstructuredConfigMap(t *testing.T, data map[string]string) *unstructured.Unstructured {
	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: configMapName,
		},
	}
	cm.Data = data
	result := &unstructured.Unstructured{}
	err := scheme.Scheme.Convert(cm, result, nil)
	if err != nil {
		t.Fatalf("Could not create unstructured ConfigMap: %v, err: %v", cm, err)
	}

	return result
}

func makeUnstructuredDeployment(t *testing.T, name string, labels map[string]string) *unstructured.Unstructured {
	return makeUnstructuredDeploymentReplicasWithLabels(t, name, 1, labels)
}

func makeUnstructuredDeploymentReplicas(t *testing.T, name string, replicas int32) *unstructured.Unstructured {
	return makeUnstructuredDeploymentReplicasWithLabels(t, name, replicas, map[string]string{})
}

func makeUnstructuredDeploymentReplicasWithLabels(t *testing.T, name string, replicas int32, labels map[string]string) *unstructured.Unstructured {
	d := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
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

func makeUnstructuredHPA(t *testing.T, name string, minReplicas int32) *unstructured.Unstructured {
	hpa := &autoscalingv2beta1.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: autoscalingv2beta1.HorizontalPodAutoscalerSpec{
			MinReplicas: &minReplicas,
		},
	}

	result := &unstructured.Unstructured{}
	err := scheme.Scheme.Convert(hpa, result, nil)
	if err != nil {
		t.Fatalf("Could not create unstructured HPA: %v, err: %v", hpa, err)
	}

	return result
}
