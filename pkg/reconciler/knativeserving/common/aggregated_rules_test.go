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

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	util "knative.dev/operator/pkg/reconciler/common/testing"
	"sigs.k8s.io/yaml"
)

func TestAggregationRuleTransform(t *testing.T) {
	tests := []struct {
		Name              string
		Input             *unstructured.Unstructured
		Existing          *unstructured.Unstructured
		OverwriteExpected bool
	}{}
	var testData = []byte(`
- name: "existing role has rules"
  input:
    kind: ClusterRole
    apiVersion: rbac.authorization.k8s.io/v1
    metadata:
      name: knative-serving-admin
    aggregationRule:
      clusterRoleSelectors:
      - matchLabels:
          serving.knative.dev/controller: "true"
    rules: []
  existing:
    kind: ClusterRole
    apiVersion: rbac.authorization.k8s.io/v1
    metadata:
      name: knative-serving-admin
    aggregationRule:
      clusterRoleSelectors:
      - matchLabels:
          serving.knative.dev/controller: "true"
    rules:
    - apiGroups:
      - serving.knative.dev
      resources:
      - services
      verbs:
      - watch
  overwriteExpected: true
- name: "no existing role"
  input:
    kind: ClusterRole
    apiVersion: rbac.authorization.k8s.io/v1
    metadata:
      name: knative-serving-admin
    aggregationRule:
      clusterRoleSelectors:
      - matchLabels:
          serving.knative.dev/controller: "true"
    rules: []
  overwriteExpected: false
`)
	err := yaml.Unmarshal(testData, &tests)
	if err != nil {
		t.Error(err)
		return
	}
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			mock := mockGetter{test.Existing}
			original := test.Input.DeepCopy()
			transformer := AggregationRuleTransform(&mock)
			if err := transformer(test.Input); err != nil {
				t.Error(err)
			}
			if test.OverwriteExpected {
				util.AssertDeepEqual(t, test.Input, test.Existing)
			} else {
				util.AssertDeepEqual(t, test.Input, original)
			}
		})
	}
}

type mockGetter struct {
	u *unstructured.Unstructured
}

func (m *mockGetter) Get(obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	if m.u == nil {
		return nil, errors.NewNotFound(schema.GroupResource{}, "")
	}
	return m.u, nil
}
