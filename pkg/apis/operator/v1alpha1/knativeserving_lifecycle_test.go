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

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var ignoreAllButTypeAndStatus = cmpopts.IgnoreFields(
	apis.Condition{},
	"LastTransitionTime", "Message", "Reason", "Severity")

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

func TestKnativeServingStatusGetCondition(t *testing.T) {
	ks := &KnativeServingStatus{}
	if a := ks.GetCondition(InstallSucceeded); a != nil {
		t.Errorf("empty ServingStatus returned %v when expected nil", a)
	}
	mc := &apis.Condition{
		Type:   InstallSucceeded,
		Status: corev1.ConditionTrue,
	}
	ks.MarkInstallSucceeded()
	if diff := cmp.Diff(mc, ks.GetCondition(InstallSucceeded), cmpopts.IgnoreFields(apis.Condition{}, "LastTransitionTime")); diff != "" {
		t.Errorf("GetCondition refs diff (-want +got): %v", diff)
	}
}

func TestKnativeServingDeploymentNotReady(t *testing.T) {
	reason := "NotReady"
	message := "Waiting on deployments"
	ks := &KnativeServingStatus{}
	mc := &apis.Condition{
		Type:    DeploymentsAvailable,
		Status:  corev1.ConditionFalse,
		Reason:  reason,
		Message: message,
	}
	ks.MarkDeploymentsNotReady()
	if diff := cmp.Diff(mc, ks.GetCondition(DeploymentsAvailable), cmpopts.IgnoreFields(apis.Condition{}, "LastTransitionTime")); diff != "" {
		t.Errorf("GetCondition refs diff (-want +got): %v", diff)
	}
}

func TestKnativeServingDeploymentsAvailable(t *testing.T) {
	ks := &KnativeServingStatus{}
	mc := &apis.Condition{
		Type:   DeploymentsAvailable,
		Status: corev1.ConditionTrue,
	}
	ks.MarkDeploymentsAvailable()
	if diff := cmp.Diff(mc, ks.GetCondition(DeploymentsAvailable), cmpopts.IgnoreFields(apis.Condition{}, "LastTransitionTime")); diff != "" {
		t.Errorf("GetCondition refs diff (-want +got): %v", diff)
	}
}

func TestKnativeServingDependenciesInstalled(t *testing.T) {
	ks := &KnativeServingStatus{}
	mc := &apis.Condition{
		Type:   DependenciesInstalled,
		Status: corev1.ConditionTrue,
	}
	ks.MarkDependenciesInstalled()
	if diff := cmp.Diff(mc, ks.GetCondition(DependenciesInstalled), cmpopts.IgnoreFields(apis.Condition{}, "LastTransitionTime")); diff != "" {
		t.Errorf("GetCondition refs diff (-want +got): %v", diff)
	}
}

func TestKnativeServingInitializeConditions(t *testing.T) {
	tests := []struct {
		name string
		ke   *KnativeServingStatus
		want *KnativeServingStatus
	}{{
		name: "empty",
		ke:   &KnativeServingStatus{},
		want: &KnativeServingStatus{
			Status: duckv1.Status{
				Conditions: []apis.Condition{{
					Type:   DependenciesInstalled,
					Status: corev1.ConditionUnknown,
				}, {
					Type:   DeploymentsAvailable,
					Status: corev1.ConditionUnknown,
				}, {
					Type:   InstallSucceeded,
					Status: corev1.ConditionUnknown,
				}, {
					Type:   "Ready",
					Status: corev1.ConditionUnknown,
				}},
			},
		},
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.ke.InitializeConditions()
			if diff := cmp.Diff(test.want, test.ke, ignoreAllButTypeAndStatus); diff != "" {
				t.Errorf("unexpected conditions (-want, +got) = %v", diff)
			}
		})
	}
}
