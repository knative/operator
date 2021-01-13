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

	"github.com/google/go-cmp/cmp"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/client-go/kubernetes/scheme"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
)

func TestPingsourceMTAadapterTransform(t *testing.T) {
	tests := []struct {
		Name     string
		Input    *unstructured.Unstructured
		Existing *unstructured.Unstructured
		Expected *unstructured.Unstructured
	}{}
	var testData = []byte(`
- name: "existing pingsource-mt-adapter has the same number of containers, but different env vars and replicas"
  input:
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: pingsource-mt-adapter
      namespace: knative-eventing
    spec:
      replicas: 0
      template:
        spec:
          containers:
            - name: dispatcher
              image: gcr.io/knative-releases/knative.dev/eventing/cmd/mtping@sha256:d6b4bd0d75a67c486f36eb34534178154db81b2ee85c0b18d7ca5269b36df037
              env:
                - name: SYSTEM_NAMESPACE
                  valueFrom:
                    fieldRef:
                      apiVersion: v1
                      fieldPath: metadata.namespace
                - name: K_METRICS_CONFIG
                  value: ''
                - name: K_LOGGING_CONFIG
                  value: ''
                - name: K_LOGGING_CONFIG_1
                  value: 'overwrite'
  existing:
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: pingsource-mt-adapter
      namespace: knative-eventing
    spec:
      replicas: 1
      template:
        spec:
          containers:
            - name: dispatcher
              image: gcr.io/knative-releases/knative.dev/eventing/cmd/mtping@sha256:d6b4bd0d75a67c486f36eb34534178154db81b2ee85c0b18d7ca5269b36df037
              env:
                - name: SYSTEM_NAMESPACE
                  valueFrom:
                    fieldRef:
                      apiVersion: v1
                      fieldPath: metadata.namespace
                - name: K_METRICS_CONFIG
                  value: 'old'
                - name: K_LOGGING_CONFIG
                  value: 'old'
                - name: K_LOGGING_CONFIG_1
                  value: 'old'
  expected:
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: pingsource-mt-adapter
      namespace: knative-eventing
    spec:
      replicas: 1
      template:
        spec:
          containers:
            - name: dispatcher
              image: gcr.io/knative-releases/knative.dev/eventing/cmd/mtping@sha256:d6b4bd0d75a67c486f36eb34534178154db81b2ee85c0b18d7ca5269b36df037
              env:
                - name: SYSTEM_NAMESPACE
                  valueFrom:
                    fieldRef:
                      apiVersion: v1
                      fieldPath: metadata.namespace
                - name: K_METRICS_CONFIG
                  value: 'old'
                - name: K_LOGGING_CONFIG
                  value: 'old'
                - name: K_LOGGING_CONFIG_1
                  value: 'overwrite'
- name: "existing pingsource-mt-adapter has less containers"
  input:
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: pingsource-mt-adapter
      namespace: knative-eventing
    spec:
      replicas: 0
      template:
        spec:
          containers:
            - name: dispatcher
              image: gcr.io/knative-releases/knative.dev/eventing/cmd/mtping@sha256:d6b4bd0d75a67c486f36eb34534178154db81b2ee85c0b18d7ca5269b36df037
              env:
                - name: SYSTEM_NAMESPACE
                  valueFrom:
                    fieldRef:
                      apiVersion: v1
                      fieldPath: metadata.namespace
                - name: K_METRICS_CONFIG
                  value: ''
                - name: K_LOGGING_CONFIG
                  value: ''
            - name: dispatcher1
              image: gcr.io/knative-releases/knative.dev/eventing/cmd/mtping@sha256:d6b4bd0d75a67c486f36eb34534178154db81b2ee85c0b18d7ca5269b36df037
              env:
                - name: SYSTEM_NAMESPACE
                  valueFrom:
                    fieldRef:
                      apiVersion: v1
                      fieldPath: metadata.namespace
                - name: K_METRICS_CONFIG
                  value: ''
                - name: K_LOGGING_CONFIG
                  value: ''
  existing:
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: pingsource-mt-adapter
      namespace: knative-eventing
    spec:
      replicas: 1
      template:
        spec:
          containers:
            - name: dispatcher
              image: gcr.io/knative-releases/knative.dev/eventing/cmd/mtping@sha256:d6b4bd0d75a67c486f36eb34534178154db81b2ee85c0b18d7ca5269b36df037
              env:
                - name: SYSTEM_NAMESPACE
                  valueFrom:
                    fieldRef:
                      apiVersion: v2
                      fieldPath: metadata.namespace
                - name: K_METRICS_CONFIG
                  value: 'test1'
                - name: K_LOGGING_CONFIG
                  value: 'test2'
  expected:
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: pingsource-mt-adapter
      namespace: knative-eventing
    spec:
      replicas: 1
      template:
        spec:
          containers:
            - name: dispatcher
              image: gcr.io/knative-releases/knative.dev/eventing/cmd/mtping@sha256:d6b4bd0d75a67c486f36eb34534178154db81b2ee85c0b18d7ca5269b36df037
              env:
                - name: SYSTEM_NAMESPACE
                  valueFrom:
                    fieldRef:
                      apiVersion: v2
                      fieldPath: metadata.namespace
                - name: K_METRICS_CONFIG
                  value: 'test1'
                - name: K_LOGGING_CONFIG
                  value: 'test2'
            - name: dispatcher1
              image: gcr.io/knative-releases/knative.dev/eventing/cmd/mtping@sha256:d6b4bd0d75a67c486f36eb34534178154db81b2ee85c0b18d7ca5269b36df037
              env:
                - name: SYSTEM_NAMESPACE
                  valueFrom:
                    fieldRef:
                      apiVersion: v1
                      fieldPath: metadata.namespace
                - name: K_METRICS_CONFIG
                  value: ''
                - name: K_LOGGING_CONFIG
                  value: ''
- name: "existing pingsource-mt-adapter has more containers"
  input:
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: pingsource-mt-adapter
      namespace: knative-eventing
    spec:
      replicas: 0
      template:
        spec:
          containers:
            - name: dispatcher
              image: gcr.io/knative-releases/knative.dev/eventing/cmd/mtping@sha256:d6b4bd0d75a67c486f36eb34534178154db81b2ee85c0b18d7ca5269b36df037
              env:
                - name: SYSTEM_NAMESPACE
                  valueFrom:
                    fieldRef:
                      apiVersion: v1
                      fieldPath: metadata.namespace
                - name: K_METRICS_CONFIG
                  value: ''
                - name: K_LOGGING_CONFIG
                  value: ''
  existing:
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: pingsource-mt-adapter
      namespace: knative-eventing
    spec:
      replicas: 1
      template:
        spec:
          containers:
            - name: dispatcher1
              image: gcr.io/knative-releases/knative.dev/eventing/cmd/mtping@sha256:d6b4bd0d75a67c486f36eb34534178154db81b2ee85c0b18d7ca5269b36df037
              env:
                - name: SYSTEM_NAMESPACE
                  valueFrom:
                    fieldRef:
                      apiVersion: v2
                      fieldPath: metadata.namespace
                - name: K_METRICS_CONFIG
                  value: 'test1'
                - name: K_LOGGING_CONFIG
                  value: 'test2'
            - name: dispatcher
              image: gcr.io/knative-releases/knative.dev/eventing/cmd/mtping@sha256:d6b4bd0d75a67c486f36eb34534178154db81b2ee85c0b18d7ca5269b36df037
              env:
                - name: SYSTEM_NAMESPACE
                  valueFrom:
                    fieldRef:
                      apiVersion: v2
                      fieldPath: metadata.namespace
                - name: K_METRICS_CONFIG
                  value: 'test1'
                - name: K_LOGGING_CONFIG
                  value: 'test2'
  expected:
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: pingsource-mt-adapter
      namespace: knative-eventing
    spec:
      replicas: 1
      template:
        spec:
          containers:
            - name: dispatcher
              image: gcr.io/knative-releases/knative.dev/eventing/cmd/mtping@sha256:d6b4bd0d75a67c486f36eb34534178154db81b2ee85c0b18d7ca5269b36df037
              env:
                - name: SYSTEM_NAMESPACE
                  valueFrom:
                    fieldRef:
                      apiVersion: v2
                      fieldPath: metadata.namespace
                - name: K_METRICS_CONFIG
                  value: 'test1'
                - name: K_LOGGING_CONFIG
                  value: 'test2'
- name: "existing pingsource-mt-adapter has the same number of containers, but with less env vars"
  input:
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: pingsource-mt-adapter
      namespace: knative-eventing
    spec:
      replicas: 0
      template:
        spec:
          containers:
            - name: dispatcher
              image: gcr.io/knative-releases/knative.dev/eventing/cmd/mtping@sha256:d6b4bd0d75a67c486f36eb34534178154db81b2ee85c0b18d7ca5269b36df037
              env:
                - name: SYSTEM_NAMESPACE
                  valueFrom:
                    fieldRef:
                      apiVersion: v1
                      fieldPath: metadata.namespace
                - name: K_METRICS_CONFIG
                  value: ''
                - name: K_LOGGING_CONFIG
                  value: 'new-env-var'
  existing:
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: pingsource-mt-adapter
      namespace: knative-eventing
    spec:
      replicas: 1
      template:
        spec:
          containers:
            - name: dispatcher
              image: gcr.io/knative-releases/knative.dev/eventing/cmd/mtping@sha256:d6b4bd0d75a67c486f36eb34534178154db81b2ee85c0b18d7ca5269b36df037
              env:
                - name: SYSTEM_NAMESPACE
                  valueFrom:
                    fieldRef:
                      apiVersion: v1
                      fieldPath: metadata.namespace
                - name: K_METRICS_CONFIG
                  value: 'test1'
  expected:
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: pingsource-mt-adapter
      namespace: knative-eventing
    spec:
      replicas: 1
      template:
        spec:
          containers:
            - name: dispatcher
              image: gcr.io/knative-releases/knative.dev/eventing/cmd/mtping@sha256:d6b4bd0d75a67c486f36eb34534178154db81b2ee85c0b18d7ca5269b36df037
              env:
                - name: SYSTEM_NAMESPACE
                  valueFrom:
                    fieldRef:
                      apiVersion: v1
                      fieldPath: metadata.namespace
                - name: K_METRICS_CONFIG
                  value: 'test1'
                - name: K_LOGGING_CONFIG
                  value: 'new-env-var'
- name: "existing pingsource-mt-adapter has the same number of containers, but with more env vars"
  input:
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: pingsource-mt-adapter
      namespace: knative-eventing
    spec:
      replicas: 0
      template:
        spec:
          containers:
            - name: dispatcher
              image: gcr.io/knative-releases/knative.dev/eventing/cmd/mtping@sha256:d6b4bd0d75a67c486f36eb34534178154db81b2ee85c0b18d7ca5269b36df037
              env:
                - name: SYSTEM_NAMESPACE
                  valueFrom:
                    fieldRef:
                      apiVersion: v1
                      fieldPath: metadata.namespace
                - name: K_METRICS_CONFIG
                  value: ''
  existing:
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: pingsource-mt-adapter
      namespace: knative-eventing
    spec:
      replicas: 1
      template:
        spec:
          containers:
            - name: dispatcher
              image: gcr.io/knative-releases/knative.dev/eventing/cmd/mtping@sha256:d6b4bd0d75a67c486f36eb34534178154db81b2ee85c0b18d7ca5269b36df037
              env:
                - name: SYSTEM_NAMESPACE
                  valueFrom:
                    fieldRef:
                      apiVersion: v1
                      fieldPath: metadata.namespace
                - name: K_METRICS_CONFIG
                  value: 'test1'
                - name: K_LOGGING_CONFIG
                  value: 'existing-env-var'
  expected:
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: pingsource-mt-adapter
      namespace: knative-eventing
    spec:
      replicas: 1
      template:
        spec:
          containers:
            - name: dispatcher
              image: gcr.io/knative-releases/knative.dev/eventing/cmd/mtping@sha256:d6b4bd0d75a67c486f36eb34534178154db81b2ee85c0b18d7ca5269b36df037
              env:
                - name: SYSTEM_NAMESPACE
                  valueFrom:
                    fieldRef:
                      apiVersion: v1
                      fieldPath: metadata.namespace
                - name: K_METRICS_CONFIG
                  value: 'test1'
                - name: K_LOGGING_CONFIG
                  value: 'existing-env-var'
- name: "existing pingsource-mt-adapter needs to delete env var"
  input:
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: pingsource-mt-adapter
      namespace: knative-eventing
    spec:
      replicas: 0
      template:
        spec:
          containers:
            - name: dispatcher
              image: gcr.io/knative-releases/knative.dev/eventing/cmd/mtping@sha256:d6b4bd0d75a67c486f36eb34534178154db81b2ee85c0b18d7ca5269b36df037
              env:
                - name: SYSTEM_NAMESPACE
                  valueFrom:
                    fieldRef:
                      apiVersion: v1
                      fieldPath: metadata.namespace
                - name: K_METRICS_CONFIG
                  value: ''
                - name: K_LOGGING_CONFIG
                  value: ''
  existing:
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: pingsource-mt-adapter
      namespace: knative-eventing
    spec:
      replicas: 1
      template:
        spec:
          containers:
            - name: dispatcher
              image: gcr.io/knative-releases/knative.dev/eventing/cmd/mtping@sha256:d6b4bd0d75a67c486f36eb34534178154db81b2ee85c0b18d7ca5269b36df037
              env:
                - name: SYSTEM_NAMESPACE
                  valueFrom:
                    fieldRef:
                      apiVersion: v1
                      fieldPath: metadata.namespace
                - name: K_METRICS_CONFIG
                  value: 'old'
                - name: K_LOGGING_CONFIG
                  value: 'old'
                - name: K_LOGGING_CONFIG_1
                  value: 'old'
  expected:
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: pingsource-mt-adapter
      namespace: knative-eventing
    spec:
      replicas: 1
      template:
        spec:
          containers:
            - name: dispatcher
              image: gcr.io/knative-releases/knative.dev/eventing/cmd/mtping@sha256:d6b4bd0d75a67c486f36eb34534178154db81b2ee85c0b18d7ca5269b36df037
              env:
                - name: SYSTEM_NAMESPACE
                  valueFrom:
                    fieldRef:
                      apiVersion: v1
                      fieldPath: metadata.namespace
                - name: K_METRICS_CONFIG
                  value: 'old'
                - name: K_LOGGING_CONFIG
                  value: 'old'
`)
	err := yaml.Unmarshal(testData, &tests)
	if err != nil {
		t.Error(err)
		return
	}
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			mock := mockGetter{test.Existing}
			transformer := ReplicasEnvVarsTransform(&mock)
			if err := transformer(test.Input); err != nil {
				t.Error(err)
			}
			got := &appsv1.Deployment{}
			if err := scheme.Scheme.Convert(test.Input, got, nil); err != nil {
				t.Fatal(err)
			}

			want := &appsv1.Deployment{}
			if err := scheme.Scheme.Convert(test.Expected, want, nil); err != nil {
				t.Fatal(err)
			}

			if !cmp.Equal(got, want) {
				t.Errorf("Not equal: (+got, -want):\n%s", cmp.Diff(got, want))
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
