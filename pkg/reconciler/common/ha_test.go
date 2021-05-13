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

	v1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
	util "knative.dev/operator/pkg/reconciler/common/testing"
)

func TestHighAvailabilityTransform(t *testing.T) {
	cases := []struct {
		name     string
		config   *v1alpha1.HighAvailability
		version  string
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
		config: makeHa(2),
		in:     makeUnstructuredConfigMap(t, nil),
		expected: makeUnstructuredConfigMap(t, map[string]string{
			enabledComponentsKey: servingComponentsValue}),
	}, {
		name:     "HA; controller",
		config:   makeHa(2),
		in:       makeUnstructuredDeployment(t, "controller"),
		expected: makeUnstructuredDeploymentReplicas(t, "controller", 2),
	}, {
		name:     "HA; autoscaler after v0.19",
		config:   makeHa(2),
		version:  "0.19.0",
		in:       makeUnstructuredDeployment(t, "autoscaler"),
		expected: makeUnstructuredDeploymentReplicas(t, "autoscaler", 2),
	}, {
		name:     "HA; autoscaler before v0.19",
		config:   makeHa(2),
		version:  "0.18.2",
		in:       makeUnstructuredDeployment(t, "autoscaler"),
		expected: makeUnstructuredDeploymentReplicas(t, "autoscaler", 1), // autoscaler HA is not supported before serving v0.19.
	}, {
		name:     "HA; autoscaler-hpa",
		config:   makeHa(2),
		in:       makeUnstructuredDeployment(t, "autoscaler-hpa"),
		expected: makeUnstructuredDeploymentReplicas(t, "autoscaler-hpa", 2),
	}, {
		name:     "HA; networking-certmanager",
		config:   makeHa(2),
		in:       makeUnstructuredDeployment(t, "networking-certmanager"),
		expected: makeUnstructuredDeploymentReplicas(t, "networking-certmanager", 2),
	}, {
		name:     "HA; networking-ns-cert",
		config:   makeHa(2),
		in:       makeUnstructuredDeployment(t, "networking-ns-cert"),
		expected: makeUnstructuredDeploymentReplicas(t, "networking-ns-cert", 2),
	}, {
		name:     "HA; networking-istio",
		config:   makeHa(2),
		in:       makeUnstructuredDeployment(t, "networking-istio"),
		expected: makeUnstructuredDeploymentReplicas(t, "networking-istio", 2),
	}, {
		name:     "HA; some-unsupported-controller",
		config:   makeHa(2),
		in:       makeUnstructuredDeployment(t, "some-unsupported-controller"),
		expected: makeUnstructuredDeployment(t, "some-unsupported-controller"),
	}, {
		name:     "HA; adjust hpa",
		config:   makeHa(2),
		in:       makeUnstructuredHPA(t, "activator", 1),
		expected: makeUnstructuredHPA(t, "activator", 2),
	}, {
		name:     "HA; keep higher hpa value",
		config:   makeHa(2),
		in:       makeUnstructuredHPA(t, "activator", 3),
		expected: makeUnstructuredHPA(t, "activator", 3),
	}, {
		name:     "HA; pingsource-mt-adapter",
		config:   makeHa(2),
		in:       makeUnstructuredDeployment(t, "pingsource-mt-adapter"),
		expected: makeUnstructuredDeploymentReplicas(t, "pingsource-mt-adapter", 2),
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {

			instance := &v1alpha1.KnativeServing{
				Spec: v1alpha1.KnativeServingSpec{
					CommonSpec: v1alpha1.CommonSpec{
						HighAvailability: tc.config,
					},
				},
			}
			instance.Status.SetVersion(tc.version)

			haTransform := HighAvailabilityTransform(instance, log)
			err := haTransform(tc.in)

			util.AssertDeepEqual(t, err, tc.err)
			util.AssertDeepEqual(t, tc.in, tc.expected)
		})
	}
}

func makeHa(replicas int32) *v1alpha1.HighAvailability {
	return &v1alpha1.HighAvailability{
		Replicas: replicas,
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
