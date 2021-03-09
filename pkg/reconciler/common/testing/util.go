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
package testing

import (
	"reflect"
	"testing"

	mf "github.com/manifestival/manifestival"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
)

func MakeDeployment(name string, podSpec corev1.PodSpec) *appsv1.Deployment {
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind: "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: podSpec,
			},
		},
	}
}

func MakeDaemonSet(name string, podSpec corev1.PodSpec) *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			Kind: "DaemonSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: appsv1.DaemonSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: podSpec,
			},
		},
	}
}

func MakeUnstructured(t *testing.T, obj interface{}) unstructured.Unstructured {
	t.Helper()
	var result = unstructured.Unstructured{}
	err := scheme.Scheme.Convert(obj, &result, nil)
	if err != nil {
		t.Fatalf("Could not create unstructured object: %v, err: %v", result, err)
	}
	return result
}

func AssertEqual(t *testing.T, actual, expected interface{}) {
	t.Helper()
	if actual == expected {
		return
	}
	t.Fatalf("expected does not equal actual. \nExpected: %v\nActual: %v", expected, actual)
}

func AssertDeepEqual(t *testing.T, actual, expected interface{}) {
	t.Helper()
	if reflect.DeepEqual(actual, expected) {
		return
	}
	t.Fatalf("expected does not deep equal actual. \nExpected: %T %+v\nActual:   %T %+v", expected, expected, actual, actual)
}

// ResourceMatchWithPath returns true if the resources in the actual manifest match the same resources in
// the expected manifest of the expected path, in terms of name, namespace, group and kind. The number of
// resources does not necessarily have to match.
func ResourceMatchWithPath(actual mf.Manifest, expectedManifestPath string) bool {
	if expectedManifestPath == "" && len(actual.Resources()) == 0 {
		return true
	}
	expected, err := mf.NewManifest(expectedManifestPath)
	if err != nil {
		return false
	}
	return resourceMatch(actual, expected)
}

func resourceMatch(actual, expected mf.Manifest) bool {
	// The resource match in terms of name, namespace, kind and group.
	if len(actual.Filter(mf.Not(mf.In(expected))).Resources()) != 0 {
		return false
	}
	if len(expected.Filter(mf.Not(mf.In(actual))).Resources()) != 0 {
		return false
	}
	return true
}

// DeepMatchWithPath returns true if the resources in the actual manifest match exactly the same resources in
// the expected manifest of the expected path. Two manifests are identical, in terms of every attribute. The number of the resources
// has to match.
func DeepMatchWithPath(actual mf.Manifest, expectedManifestPath string) bool {
	if expectedManifestPath == "" && len(actual.Resources()) == 0 {
		return true
	}
	expected, err := mf.NewManifest(expectedManifestPath)
	if err != nil {
		return false
	}
	return resourceDeepMatch(actual, expected)
}

func resourceDeepMatch(actual, expected mf.Manifest) bool {
	if len(expected.Resources()) != len(actual.Resources()) {
		return false
	}

	if !resourceMatch(actual, expected) {
		return false
	}
	return manifestCompare(actual, expected)
}

// ResourceContainingWithPath returns true if the resources in the actual manifest contains all the resources in
// the expected manifest of the expected path, but the number of resources is not necessarily the same. For the
// identical resources, they need to match in terms of every attribute.
func ResourceContainingWithPath(actual mf.Manifest, expectedManifestPath string) bool {
	expected, err := mf.NewManifest(expectedManifestPath)
	if err != nil {
		return false
	}
	return resourceContaining(actual, expected)
}

func resourceContaining(actual, expected mf.Manifest) bool {
	if len(expected.Resources()) > len(actual.Resources()) {
		return false
	}

	// All resources in the expected exist in the actual manifest, but the actual may contain more.
	if len(expected.Filter(mf.Not(mf.In(actual))).Resources()) != 0 {
		return false
	}

	return manifestCompare(actual, expected)
}

func manifestCompare(actual, expected mf.Manifest) bool {
	for _, expectedU := range expected.Resources() {
		match := false
		for _, actualU := range actual.Resources() {
			if equality.Semantic.DeepEqual(actualU, expectedU) {
				// If we find the matched resource, stop the iteration for this resource.
				match = true
				break
			}
		}
		// When one expected resource has finished the checking, we know whether a match is found or not.
		if !match {
			return false
		}
	}
	return true
}
