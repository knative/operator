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
package v1alpha1

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestKnativeServingGroupVersionKind(t *testing.T) {
	r := &KnativeServing{}
	want := schema.GroupVersionKind{
		Group:   GroupName,
		Version: SchemaVersion,
		Kind:    Kind,
	}
	if got := r.GroupVersionKind(); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

func TestAssumeDepsInstalled(t *testing.T) {
	ks := &KnativeServingStatus{}
	ks.InitializeConditions()
	assertEqual(t, ks.GetCondition(DependenciesInstalled).IsUnknown(), true)
	assertEqual(t, ks.GetCondition(DependenciesInstalled).IsTrue(), false)
	ks.MarkInstallSucceeded()
	assertEqual(t, ks.GetCondition(DependenciesInstalled).IsUnknown(), false)
	assertEqual(t, ks.GetCondition(DependenciesInstalled).IsTrue(), true)
	assertEqual(t, ks.IsInstalled(), true)
	assertEqual(t, ks.IsFullySupported(), true)
}

func assertEqual(t *testing.T, actual, expected interface{}) {
	if actual == expected {
		return
	}
	t.Fatalf("Expected: %v\nActual: %v", expected, actual)
}
