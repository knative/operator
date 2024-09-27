/*
Copyright 2024 The Knative Authors

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

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"

	"knative.dev/operator/pkg/apis/operator/base"
)

func TestNamespaceConfigurationTransform(t *testing.T) {
	tests := []struct {
		name           string
		namespace      *corev1.Namespace
		override       *base.NamespaceConfiguration
		expLabels      map[string]string
		expAnnotations map[string]string
	}{{
		name: "Label override",
		namespace: &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "knative-serving",
				Labels: map[string]string{
					"istio-injection":             "enabled",
					"serving.knative.dev/release": "v0.13.0",
				},
			},
		},
		override: &base.NamespaceConfiguration{
			Labels: map[string]string{"a": "b"},
		},
		expLabels:      map[string]string{"a": "b", "istio-injection": "enabled", "serving.knative.dev/release": "v0.13.0"},
		expAnnotations: nil,
	}, {
		name: "Annotation override",
		namespace: &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "knative-serving",
				Labels: map[string]string{
					"istio-injection":             "enabled",
					"serving.knative.dev/release": "v0.13.0",
				},
			},
		},
		override: &base.NamespaceConfiguration{
			Annotations: map[string]string{"c": "d"},
		},
		expLabels:      map[string]string{"istio-injection": "enabled", "serving.knative.dev/release": "v0.13.0"},
		expAnnotations: map[string]string{"c": "d"},
	}, {
		name: "No override",
		namespace: &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "knative-serving",
				Labels: map[string]string{
					"istio-injection":             "enabled",
					"serving.knative.dev/release": "v0.13.0",
				},
			},
		},
		override:       nil,
		expLabels:      map[string]string{"serving.knative.dev/release": "v0.13.0", "istio-injection": "enabled"},
		expAnnotations: nil,
	}, {
		name: "Override both labels and annotations",
		namespace: &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "knative-serving",
			},
		},
		override: &base.NamespaceConfiguration{
			Labels:      map[string]string{"c1": "d1", "j": "k"},
			Annotations: map[string]string{"c": "d", "x": "y"},
		},
		expLabels:      map[string]string{"j": "k", "c1": "d1"},
		expAnnotations: map[string]string{"x": "y", "c": "d"},
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			u := &unstructured.Unstructured{}
			err := scheme.Scheme.Convert(test.namespace, u, nil)
			if err != nil {
				t.Fatalf("Failed to convert namespace to unstructured: %v", err)
			}
			NamespaceConfigurationTransform(test.override)(u)
			got := &corev1.Namespace{}
			if err = scheme.Scheme.Convert(u, got, nil); err != nil {
				t.Fatalf("Failed to convert unstructured to namespace: %v", err)
			}

			if diff := cmp.Diff(got.GetLabels(), test.expLabels); diff != "" {
				t.Fatalf("Unexpected labels: %v", diff)
			}

			if diff := cmp.Diff(got.GetAnnotations(), test.expAnnotations); diff != "" {
				t.Fatalf("Unexpected annotations: %v", diff)
			}
		})
	}
}
