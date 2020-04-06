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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"

	servingv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
)

func TestHighAvailabilityTransform(t *testing.T) {
	cases := []struct {
		name     string
		config   *servingv1alpha1.HighAvailability
		in       *unstructured.Unstructured
		expected *unstructured.Unstructured
		err      error
	}{
		{
			name:     "No HA; ConfigMap",
			config:   nil,
			in:       makeUnstructuredConfigMap(t, nil),
			expected: makeUnstructuredConfigMap(t, nil),
		},
		{
			name:   "HA; ConfigMap",
			config: makeHa(2),
			in:     makeUnstructuredConfigMap(t, nil),
			expected: makeUnstructuredConfigMap(t, map[string]string{
				enabledComponentsKey: componentsValue,
			}),
		},
		{
			name:     "HA; controller",
			config:   makeHa(2),
			in:       makeUnstructuredDeployment(t, "controller"),
			expected: makeUnstructuredDeploymentReplicas(t, "controller", 2),
		},
		{
			name:     "HA; autoscaler-hpa",
			config:   makeHa(2),
			in:       makeUnstructuredDeployment(t, "autoscaler-hpa"),
			expected: makeUnstructuredDeploymentReplicas(t, "autoscaler-hpa", 2),
		},
		{
			name:     "HA; networking-certmanager",
			config:   makeHa(2),
			in:       makeUnstructuredDeployment(t, "networking-certmanager"),
			expected: makeUnstructuredDeploymentReplicas(t, "networking-certmanager", 2),
		},
		{
			name:     "HA; networking-ns-cert",
			config:   makeHa(2),
			in:       makeUnstructuredDeployment(t, "networking-ns-cert"),
			expected: makeUnstructuredDeploymentReplicas(t, "networking-ns-cert", 2),
		},
		{
			name:     "HA; networking-istio",
			config:   makeHa(2),
			in:       makeUnstructuredDeployment(t, "networking-istio"),
			expected: makeUnstructuredDeploymentReplicas(t, "networking-istio", 2),
		},
		{
			name:     "HA; some-unsupported-controller",
			config:   makeHa(2),
			in:       makeUnstructuredDeployment(t, "some-unsupported-controller"),
			expected: makeUnstructuredDeployment(t, "some-unsupported-controller"),
		},
	}

	for i := range cases {
		tc := cases[i]

		instance := &servingv1alpha1.KnativeServing{
			Spec: servingv1alpha1.KnativeServingSpec{
				HighAvailability: tc.config,
			},
		}

		haTransform := HighAvailabilityTransform(instance, log)
		err := haTransform(tc.in)

		assertDeepEqual(t, err, tc.err)
		assertDeepEqual(t, tc.in, tc.expected)
	}
}

func makeHa(replicas int32) *servingv1alpha1.HighAvailability {
	return &servingv1alpha1.HighAvailability{
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
		t.Fatalf("Could not creat unstructured ConfigMap: %v, err: %v", cm, err)
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
		t.Fatalf("Could not creat unstructured Deployment: %v, err: %v", d, err)
	}

	return result
}
