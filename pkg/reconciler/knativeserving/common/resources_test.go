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
	"reflect"
	"testing"

	mf "github.com/manifestival/manifestival"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	servingv1alpha1 "knative.dev/operator/pkg/apis/serving/v1alpha1"
	"sigs.k8s.io/yaml"
)

type resourceOverrideTest struct {
	Input    servingv1alpha1.KnativeServing
	Expected map[string]v1.ResourceRequirements
}

var testdata = []byte(`
- input:
    apiVersion: operator.knative.dev/v1alpha1
    kind: KnativeServing
    metadata:
      name: no-overrides
  expected:
    activator:
      requests:
        cpu: 300m
        memory: 60Mi
      limits:
        cpu: 1
        memory: 600Mi
    autoscaler:
      requests:
        cpu: 30m
        memory: 40Mi
      limits:
        cpu: 300m
        memory: 400Mi
    controller:
      requests:
        cpu: 100m
        memory: 100Mi
      limits:
        cpu: 1
        memory: 1000Mi
    webhook:
      requests:
        cpu: 20m
        memory: 20Mi
      limits:
        cpu: 200m
        memory: 200Mi
    autoscaler-hpa:
      requests:
        cpu: 30m
        memory: 40Mi
      limits:
        cpu: 300m
        memory: 400Mi
    networking-istio:
      requests:
        cpu: 30m
        memory: 40Mi
      limits:
        cpu: 300m
        memory: 400Mi
- input:
    apiVersion: operator.knative.dev/v1alpha1
    kind: KnativeServing
    metadata:
      name: single-container
    spec:
      resources:
      - container: activator
        limits:
          cpu: 9999m
          memory: 999Mi
  expected:
    activator:
      requests:
        cpu: 300m
        memory: 60Mi
      limits:
        cpu: 9999m
        memory: 999Mi
    autoscaler:
      requests:
        cpu: 30m
        memory: 40Mi
      limits:
        cpu: 300m
        memory: 400Mi
    controller:
      requests:
        cpu: 100m
        memory: 100Mi
      limits:
        cpu: 1
        memory: 1000Mi
    webhook:
      requests:
        cpu: 20m
        memory: 20Mi
      limits:
        cpu: 200m
        memory: 200Mi
    autoscaler-hpa:
      requests:
        cpu: 30m
        memory: 40Mi
      limits:
        cpu: 300m
        memory: 400Mi
    networking-istio:
      requests:
        cpu: 30m
        memory: 40Mi
      limits:
        cpu: 300m
        memory: 400Mi
- input:
    apiVersion: operator.knative.dev/v1alpha1
    kind: KnativeServing
    metadata:
      name: multi-container
    spec:
      resources:
      - container: webhook
        requests:
          cpu: 22m
          memory: 22Mi
        limits:
          cpu: 220m
          memory: 220Mi
      - container: another
        requests:
          cpu: 33m
          memory: 42Mi
        limits:
          cpu: 330m
          memory: 420Mi
  expected:
    webhook:
      requests:
        cpu: 22m
        memory: 22Mi
      limits:
        cpu: 220m
        memory: 220Mi
    another:
      requests:
        cpu: 33m
        memory: 42Mi
      limits:
        cpu: 330m
        memory: 420Mi
    activator:
      requests:
        cpu: 300m
        memory: 60Mi
      limits:
        cpu: 1
        memory: 600Mi
    autoscaler:
      requests:
        cpu: 30m
        memory: 40Mi
      limits:
        cpu: 300m
        memory: 400Mi
    controller:
      requests:
        cpu: 100m
        memory: 100Mi
      limits:
        cpu: 1
        memory: 1000Mi
    autoscaler-hpa:
      requests:
        cpu: 30m
        memory: 40Mi
      limits:
        cpu: 300m
        memory: 400Mi
    networking-istio:
      requests:
        cpu: 30m
        memory: 40Mi
      limits:
        cpu: 300m
        memory: 400Mi
- input:
    apiVersion: operator.knative.dev/v1alpha1
    kind: KnativeServing
    metadata:
      name: multi-deployment
    spec:
      resources:
      - container: autoscaler
        requests:
          cpu: 33m
          memory: 42Mi
        limits:
          cpu: 330m
          memory: 420Mi
      - container: controller
        requests:
          cpu: 999m
          memory: 999Mi
        limits:
          cpu: 9990m
          memory: 9990Mi
  expected:
    autoscaler:
      requests:
        cpu: 33m
        memory: 42Mi
      limits:
        cpu: 330m
        memory: 420Mi
    controller:
      requests:
        cpu: 999m
        memory: 999Mi
      limits:
        cpu: 9990m
        memory: 9990Mi
    activator:
      requests:
        cpu: 300m
        memory: 60Mi
      limits:
        cpu: 1
        memory: 600Mi
    webhook:
      requests:
        cpu: 20m
        memory: 20Mi
      limits:
        cpu: 200m
        memory: 200Mi
    autoscaler-hpa:
      requests:
        cpu: 30m
        memory: 40Mi
      limits:
        cpu: 300m
        memory: 400Mi
    networking-istio:
      requests:
        cpu: 30m
        memory: 40Mi
      limits:
        cpu: 300m
        memory: 400Mi
`)

func TestResourceRequirementsTransform(t *testing.T) {
	tests := []resourceOverrideTest{}
	err := yaml.Unmarshal(testdata, &tests)
	if err != nil {
		t.Error(err)
		return
	}
	for _, test := range tests {
		t.Run(test.Input.Name, func(t *testing.T) {
			runResourceRequirementsTransformTest(t, &test)
		})
	}
}

func runResourceRequirementsTransformTest(t *testing.T, test *resourceOverrideTest) {
	manifest, err := mf.NewManifest("testdata/manifest.yaml")
	if err != nil {
		t.Error(err)
	}
	actual, err := manifest.Transform(ResourceRequirementsTransform(&test.Input, log))
	if err != nil {
		t.Error(err)
	}
	for _, u := range actual.Filter(mf.ByKind("Deployment")).Resources() {
		deployment := &appsv1.Deployment{}
		if err := scheme.Scheme.Convert(&u, deployment, nil); err != nil {
			t.Error(err)
		}
		containers := deployment.Spec.Template.Spec.Containers
		for i := range containers {
			expected := test.Expected[containers[i].Name]
			if !reflect.DeepEqual(containers[i].Resources, expected) {
				t.Errorf("\n    Name: %s\n  Expect: %v\n  Actual: %v", containers[i].Name, expected, containers[i].Resources)
			}
		}
	}
}
