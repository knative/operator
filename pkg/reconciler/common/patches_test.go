/*
Copyright 2026 The Knative Authors

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
	"context"
	"strings"
	"testing"

	mf "github.com/manifestival/manifestival"
	mffake "github.com/manifestival/manifestival/fake"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"knative.dev/operator/pkg/apis/operator/base"
	"knative.dev/operator/pkg/apis/operator/v1beta1"
	apistest "knative.dev/pkg/apis/testing"
)

func TestApplyResourcePatchesAppliesAllTypesInOrder(t *testing.T) {
	manifest := manifestForPatchTest(t, deploymentForPatchTest(t, "activator", "knative-serving", 1))
	patches := []base.ResourcePatch{
		resourcePatchForTest(base.JSONPatchType, "apps/v1", "Deployment", "activator", `
- op: add
  path: /metadata/annotations
  value:
    example.com/json: applied
`),
		resourcePatchForTest(base.MergePatchType, "apps/v1", "Deployment", "activator", `
metadata:
  labels:
    example.com/merge: applied
spec:
  replicas: 2
`),
		resourcePatchForTest(base.StrategicMergePatchType, "apps/v1", "Deployment", "activator", `
spec:
  replicas: 3
`),
	}

	if err := ApplyResourcePatches(&manifest, patches); err != nil {
		t.Fatalf("ApplyResourcePatches() = %v", err)
	}
	got := manifest.Resources()[0]
	if value := got.GetAnnotations()["example.com/json"]; value != "applied" {
		t.Fatalf("JSON patch annotation = %q, want applied", value)
	}
	if value := got.GetLabels()["example.com/merge"]; value != "applied" {
		t.Fatalf("merge patch label = %q, want applied", value)
	}
	if replicas, _, err := unstructured.NestedInt64(got.Object, "spec", "replicas"); err != nil || replicas != 3 {
		t.Fatalf("strategic merge patch replicas = %d, %v, want 3, nil", replicas, err)
	}
}

func TestApplyResourcePatchesAppliesJSONReplaceOperation(t *testing.T) {
	manifest := manifestForPatchTest(t, deploymentForPatchTest(t, "activator", "knative-serving", 1))
	patch := resourcePatchForTest(base.JSONPatchType, "apps/v1", "Deployment", "activator", `
- op: replace
  path: /spec/replicas
  value: 4
`)

	if err := ApplyResourcePatches(&manifest, []base.ResourcePatch{patch}); err != nil {
		t.Fatalf("ApplyResourcePatches() = %v", err)
	}
	got := manifest.Resources()[0]
	if replicas, found, err := unstructured.NestedInt64(got.Object, "spec", "replicas"); err != nil || !found || replicas != 4 {
		t.Fatalf("spec.replicas = %d, found = %v, err = %v, want 4, true, nil", replicas, found, err)
	}
}

func TestApplyResourcePatchesAllowsPatchInJSONPointerPath(t *testing.T) {
	configMap := resourceForPatchTest("v1", "ConfigMap", "config-features", "knative-serving")
	if err := unstructured.SetNestedStringMap(configMap.Object, map[string]string{}, "data"); err != nil {
		t.Fatalf("SetNestedStringMap() = %v", err)
	}
	manifest := manifestForPatchTest(t, configMap)
	patch := resourcePatchForTest(base.JSONPatchType, "v1", "ConfigMap", "config-features", `
- op: add
  path: /data/$patch
  value: literal-data
`)

	if err := ApplyResourcePatches(&manifest, []base.ResourcePatch{patch}); err != nil {
		t.Fatalf("ApplyResourcePatches() = %v", err)
	}
	data, found, err := unstructured.NestedStringMap(manifest.Resources()[0].Object, "data")
	if err != nil || !found || data["$patch"] != "literal-data" {
		t.Fatalf("data = %#v, found = %v, err = %v, want $patch=literal-data", data, found, err)
	}
}

func TestApplyResourcePatchesUsesNamespaceToDisambiguateTarget(t *testing.T) {
	manifest := manifestForPatchTest(t,
		deploymentForPatchTest(t, "activator", "one", 1),
		deploymentForPatchTest(t, "activator", "two", 1),
	)
	patch := resourcePatchForTest(base.MergePatchType, "apps/v1", "Deployment", "activator", `
metadata:
  labels:
    example.com/patched: "true"
`)
	patch.Target.Namespace = "two"

	if err := ApplyResourcePatches(&manifest, []base.ResourcePatch{patch}); err != nil {
		t.Fatalf("ApplyResourcePatches() = %v", err)
	}
	resources := manifest.Resources()
	if value := resources[0].GetLabels()["example.com/patched"]; value != "" {
		t.Fatalf("namespace one patch label = %q, want empty", value)
	}
	if value := resources[1].GetLabels()["example.com/patched"]; value != "true" {
		t.Fatalf("namespace two patch label = %q, want true", value)
	}
}

func TestApplyResourcePatchesDeletesResource(t *testing.T) {
	hpa := resourceForPatchTest("autoscaling/v2", "HorizontalPodAutoscaler", "activator", "knative-serving")
	deployment := deploymentForPatchTest(t, "activator", "knative-serving", 1)
	manifest := manifestForPatchTest(t, hpa, deployment)
	patch := resourcePatchForTest(base.StrategicMergePatchType, "autoscaling/v2", "HorizontalPodAutoscaler", "activator", `$patch: delete`)

	if err := ApplyResourcePatches(&manifest, []base.ResourcePatch{patch}); err != nil {
		t.Fatalf("ApplyResourcePatches() = %v", err)
	}
	resources := manifest.Resources()
	if len(resources) != 1 || resources[0].GetKind() != "Deployment" {
		t.Fatalf("resources = %#v, want only Deployment", resources)
	}
}

func TestApplyResourcePatchesDeletesNestedListItem(t *testing.T) {
	deployment := deploymentForPatchTest(t, "activator", "knative-serving", 1)
	containers := []interface{}{
		map[string]interface{}{
			"name": "activator",
			"env": []interface{}{
				map[string]interface{}{"name": "DELETE_ME", "value": "delete"},
				map[string]interface{}{"name": "KEEP_ME", "value": "keep"},
			},
		},
	}
	if err := unstructured.SetNestedSlice(deployment.Object, containers, "spec", "template", "spec", "containers"); err != nil {
		t.Fatalf("SetNestedSlice() = %v", err)
	}
	manifest := manifestForPatchTest(t, deployment)
	patch := resourcePatchForTest(base.StrategicMergePatchType, "apps/v1", "Deployment", "activator", `
spec:
  template:
    spec:
      containers:
      - name: activator
        env:
        - name: DELETE_ME
          $patch: delete
`)

	if err := ApplyResourcePatches(&manifest, []base.ResourcePatch{patch}); err != nil {
		t.Fatalf("ApplyResourcePatches() = %v", err)
	}
	gotContainers, found, err := unstructured.NestedSlice(manifest.Resources()[0].Object, "spec", "template", "spec", "containers")
	if err != nil || !found || len(gotContainers) != 1 {
		t.Fatalf("containers = %#v, found = %v, err = %v, want one container", gotContainers, found, err)
	}
	gotContainer, ok := gotContainers[0].(map[string]interface{})
	if !ok {
		t.Fatalf("container = %#v, want map[string]interface{}", gotContainers[0])
	}
	gotEnv, found, err := unstructured.NestedSlice(gotContainer, "env")
	if err != nil || !found || len(gotEnv) != 1 {
		t.Fatalf("env = %#v, found = %v, err = %v, want one environment variable", gotEnv, found, err)
	}
	gotEnvVar, ok := gotEnv[0].(map[string]interface{})
	if !ok {
		t.Fatalf("environment variable = %#v, want map[string]interface{}", gotEnv[0])
	}
	if gotEnvVar["name"] != "KEEP_ME" || gotEnvVar["value"] != "keep" {
		t.Fatalf("environment variable = %#v, want KEEP_ME=keep", gotEnvVar)
	}
}

func TestApplyResourcePatchesReplacesNestedMap(t *testing.T) {
	deployment := deploymentForPatchTest(t, "activator", "knative-serving", 1)
	podSpec := map[string]interface{}{
		"serviceAccountName": "old-service-account",
		"containers": []interface{}{
			map[string]interface{}{"name": "old", "image": "example.com/old"},
		},
	}
	if err := unstructured.SetNestedMap(deployment.Object, podSpec, "spec", "template", "spec"); err != nil {
		t.Fatalf("SetNestedMap() = %v", err)
	}
	manifest := manifestForPatchTest(t, deployment)
	patch := resourcePatchForTest(base.StrategicMergePatchType, "apps/v1", "Deployment", "activator", `
spec:
  template:
    spec:
      $patch: replace
      containers:
      - name: replacement
        image: example.com/replacement
`)

	if err := ApplyResourcePatches(&manifest, []base.ResourcePatch{patch}); err != nil {
		t.Fatalf("ApplyResourcePatches() = %v", err)
	}
	gotPodSpec, found, err := unstructured.NestedMap(manifest.Resources()[0].Object, "spec", "template", "spec")
	if err != nil || !found {
		t.Fatalf("pod spec = %#v, found = %v, err = %v, want a pod spec", gotPodSpec, found, err)
	}
	if _, found := gotPodSpec["serviceAccountName"]; found {
		t.Fatalf("pod spec = %#v, want serviceAccountName removed", gotPodSpec)
	}
	gotContainers, found, err := unstructured.NestedSlice(gotPodSpec, "containers")
	if err != nil || !found || len(gotContainers) != 1 {
		t.Fatalf("containers = %#v, found = %v, err = %v, want one container", gotContainers, found, err)
	}
	gotContainer, ok := gotContainers[0].(map[string]interface{})
	if !ok || gotContainer["name"] != "replacement" || gotContainer["image"] != "example.com/replacement" {
		t.Fatalf("container = %#v, want replacement container", gotContainers[0])
	}
}

func TestApplyResourcePatchesReplacesNestedList(t *testing.T) {
	deployment := deploymentForPatchTest(t, "activator", "knative-serving", 1)
	containers := []interface{}{
		map[string]interface{}{"name": "activator", "image": "example.com/activator"},
		map[string]interface{}{"name": "sidecar", "image": "example.com/sidecar"},
	}
	if err := unstructured.SetNestedSlice(deployment.Object, containers, "spec", "template", "spec", "containers"); err != nil {
		t.Fatalf("SetNestedSlice() = %v", err)
	}
	manifest := manifestForPatchTest(t, deployment)
	patch := resourcePatchForTest(base.StrategicMergePatchType, "apps/v1", "Deployment", "activator", `
spec:
  template:
    spec:
      containers:
      - name: replacement
        image: example.com/replacement
      - $patch: replace
`)

	if err := ApplyResourcePatches(&manifest, []base.ResourcePatch{patch}); err != nil {
		t.Fatalf("ApplyResourcePatches() = %v", err)
	}
	gotContainers, found, err := unstructured.NestedSlice(manifest.Resources()[0].Object, "spec", "template", "spec", "containers")
	if err != nil || !found || len(gotContainers) != 1 {
		t.Fatalf("containers = %#v, found = %v, err = %v, want one container", gotContainers, found, err)
	}
	gotContainer, ok := gotContainers[0].(map[string]interface{})
	if !ok || gotContainer["name"] != "replacement" || gotContainer["image"] != "example.com/replacement" {
		t.Fatalf("container = %#v, want replacement container", gotContainers[0])
	}
}

func TestApplyResourcePatchesReplacesResourceAndPreservesIdentity(t *testing.T) {
	deployment := deploymentForPatchTest(t, "activator", "knative-serving", 1)
	deployment.SetLabels(map[string]string{"example.com/old": "label"})
	deployment.SetAnnotations(map[string]string{"example.com/old": "annotation"})
	if err := unstructured.SetNestedField(deployment.Object, true, "spec", "paused"); err != nil {
		t.Fatalf("SetNestedField() = %v", err)
	}
	manifest := manifestForPatchTest(t, deployment)
	patch := resourcePatchForTest(base.StrategicMergePatchType, "apps/v1", "Deployment", "activator", `
$patch: replace
metadata:
  labels:
    example.com/replacement: applied
spec:
  replicas: 3
`)

	if err := ApplyResourcePatches(&manifest, []base.ResourcePatch{patch}); err != nil {
		t.Fatalf("ApplyResourcePatches() = %v", err)
	}
	got := manifest.Resources()[0]
	if got.GetAPIVersion() != "apps/v1" || got.GetKind() != "Deployment" || got.GetName() != "activator" || got.GetNamespace() != "knative-serving" {
		t.Fatalf("identity = %s, %s, %s/%s, want apps/v1, Deployment, knative-serving/activator", got.GetAPIVersion(), got.GetKind(), got.GetNamespace(), got.GetName())
	}
	if labels := got.GetLabels(); len(labels) != 1 || labels["example.com/replacement"] != "applied" {
		t.Fatalf("labels = %#v, want only replacement label", labels)
	}
	if annotations := got.GetAnnotations(); len(annotations) != 0 {
		t.Fatalf("annotations = %#v, want none", annotations)
	}
	if _, found, err := unstructured.NestedFieldNoCopy(got.Object, "spec", "paused"); err != nil || found {
		t.Fatalf("spec.paused found = %v, err = %v, want false, nil", found, err)
	}
	if replicas, found, err := unstructured.NestedInt64(got.Object, "spec", "replicas"); err != nil || !found || replicas != 3 {
		t.Fatalf("spec.replicas = %d, found = %v, err = %v, want 3, true, nil", replicas, found, err)
	}
}

func TestResourcePatchStagesDeleteLiveResourceWithoutManifestStatus(t *testing.T) {
	client := mffake.New()
	hpa := resourceForPatchTest("autoscaling/v2", "HorizontalPodAutoscaler", "activator", "knative-serving")
	manifest, err := mf.ManifestFrom(mf.Slice([]unstructured.Unstructured{hpa}), mf.UseClient(client))
	if err != nil {
		t.Fatalf("ManifestFrom() = %v", err)
	}
	if err := manifest.Apply(); err != nil {
		t.Fatalf("initial Apply() = %v", err)
	}
	instance := &v1beta1.KnativeServing{
		Spec: v1beta1.KnativeServingSpec{CommonSpec: base.CommonSpec{
			Patches: []base.ResourcePatch{
				resourcePatchForTest(base.StrategicMergePatchType, "autoscaling/v2", "HorizontalPodAutoscaler", "activator", `$patch: delete`),
			},
		}},
	}

	patchStages := NewResourcePatchStages()
	if err := patchStages.Apply(context.Background(), &manifest, instance); err != nil {
		t.Fatalf("ResourcePatchStages.Apply() = %v", err)
	}
	if _, err := client.Get(&hpa); err != nil {
		t.Fatalf("Get() before deferred delete = %v", err)
	}
	if err := patchStages.Delete(context.Background(), &manifest, instance); err != nil {
		t.Fatalf("ResourcePatchStages.Delete() = %v", err)
	}
	if _, err := client.Get(&hpa); !apierrors.IsNotFound(err) {
		t.Fatalf("Get() after delete = %v, want NotFound", err)
	}
}

func TestDeletedResourceIsPrunedAndCanBeRecreated(t *testing.T) {
	client := mffake.New()
	hpa := resourceForPatchTest("autoscaling/v2", "HorizontalPodAutoscaler", "activator", "knative-serving")
	installed, err := mf.ManifestFrom(mf.Slice([]unstructured.Unstructured{hpa}), mf.UseClient(client))
	if err != nil {
		t.Fatalf("ManifestFrom() = %v", err)
	}
	if err := installed.Apply(); err != nil {
		t.Fatalf("initial Apply() = %v", err)
	}

	desired := installed.Append()
	patch := resourcePatchForTest(base.StrategicMergePatchType, "autoscaling/v2", "HorizontalPodAutoscaler", "activator", `$patch: delete`)
	if err := ApplyResourcePatches(&desired, []base.ResourcePatch{patch}); err != nil {
		t.Fatalf("ApplyResourcePatches() = %v", err)
	}
	instance := &v1beta1.KnativeServing{Status: v1beta1.KnativeServingStatus{Manifests: []string{"installed"}}}
	deleteObsolete := DeleteObsoleteResources(context.Background(), instance, func(context.Context, base.KComponent) (*mf.Manifest, error) {
		return &installed, nil
	})
	if err := deleteObsolete(context.Background(), &desired, instance); err != nil {
		t.Fatalf("DeleteObsoleteResources() = %v", err)
	}
	if _, err := client.Get(&hpa); !apierrors.IsNotFound(err) {
		t.Fatalf("Get() after delete = %v, want NotFound", err)
	}

	if err := installed.Apply(); err != nil {
		t.Fatalf("Apply() after removing the patch = %v", err)
	}
	if _, err := client.Get(&hpa); err != nil {
		t.Fatalf("Get() after recreate = %v", err)
	}
}

func TestApplyResourcePatchesValidatesTargetAndIdentity(t *testing.T) {
	tests := []struct {
		name            string
		resources       []unstructured.Unstructured
		patch           base.ResourcePatch
		targetNamespace string
		wantError       string
	}{
		{
			name:      "target does not exist",
			resources: []unstructured.Unstructured{deploymentForPatchTest(t, "controller", "knative-serving", 1)},
			patch:     resourcePatchForTest(base.MergePatchType, "apps/v1", "Deployment", "activator", `{}`),
			wantError: "did not match any generated resource",
		},
		{
			name: "target ambiguity is reported before patch errors",
			resources: []unstructured.Unstructured{
				deploymentForPatchTest(t, "activator", "one", 1),
				deploymentForPatchTest(t, "activator", "two", 1),
			},
			patch:     resourcePatchForTest(base.JSONPatchType, "apps/v1", "Deployment", "activator", `not: an-array`),
			wantError: "specify target.namespace",
		},
		{
			name: "duplicate target in one namespace",
			resources: []unstructured.Unstructured{
				deploymentForPatchTest(t, "activator", "knative-serving", 1),
				deploymentForPatchTest(t, "activator", "knative-serving", 2),
			},
			patch:           resourcePatchForTest(base.MergePatchType, "apps/v1", "Deployment", "activator", `{}`),
			targetNamespace: "knative-serving",
			wantError:       "target must match exactly one resource",
		},
		{
			name:      "identity changes",
			resources: []unstructured.Unstructured{deploymentForPatchTest(t, "activator", "knative-serving", 1)},
			patch: resourcePatchForTest(base.MergePatchType, "apps/v1", "Deployment", "activator", `
metadata:
  name: changed
`),
			wantError: "must not change apiVersion, kind, metadata.name, or metadata.namespace",
		},
		{
			name:      "root replacement changes identity",
			resources: []unstructured.Unstructured{deploymentForPatchTest(t, "activator", "knative-serving", 1)},
			patch: resourcePatchForTest(base.StrategicMergePatchType, "apps/v1", "Deployment", "activator", `
$patch: replace
metadata:
  name: changed
`),
			wantError: "must not change apiVersion, kind, metadata.name, or metadata.namespace",
		},
		{
			name:      "strategic merge unsupported for custom resource",
			resources: []unstructured.Unstructured{resourceForPatchTest("example.dev/v1", "Example", "sample", "knative-serving")},
			patch:     resourcePatchForTest(base.StrategicMergePatchType, "example.dev/v1", "Example", "sample", `spec: {enabled: true}`),
			wantError: "use a json or merge patch",
		},
		{
			name:      "CRD deletion is rejected",
			resources: []unstructured.Unstructured{resourceForPatchTest("apiextensions.k8s.io/v1", "CustomResourceDefinition", "examples.example.dev", "")},
			patch:     resourcePatchForTest(base.StrategicMergePatchType, "apiextensions.k8s.io/v1", "CustomResourceDefinition", "examples.example.dev", `$patch: delete`),
			wantError: "deleting CustomResourceDefinitions is not supported",
		},
		{
			name:      "Namespace deletion is rejected",
			resources: []unstructured.Unstructured{resourceForPatchTest("v1", "Namespace", "knative-serving", "")},
			patch:     resourcePatchForTest(base.StrategicMergePatchType, "v1", "Namespace", "knative-serving", `$patch: delete`),
			wantError: "deleting Namespaces is not supported",
		},
		{
			name:      "root $patch directive rejected for merge patch",
			resources: []unstructured.Unstructured{deploymentForPatchTest(t, "activator", "knative-serving", 1)},
			patch:     resourcePatchForTest(base.MergePatchType, "apps/v1", "Deployment", "activator", `$patch: delete`),
			wantError: "$patch directive is only supported for the strategic merge patch type",
		},
		{
			name:      "root $patch directive rejected for json patch",
			resources: []unstructured.Unstructured{deploymentForPatchTest(t, "activator", "knative-serving", 1)},
			patch:     resourcePatchForTest(base.JSONPatchType, "apps/v1", "Deployment", "activator", `$patch: delete`),
			wantError: "$patch directive is only supported for the strategic merge patch type",
		},
		{
			name:      "root replacement rejected for merge patch",
			resources: []unstructured.Unstructured{deploymentForPatchTest(t, "activator", "knative-serving", 1)},
			patch:     resourcePatchForTest(base.MergePatchType, "apps/v1", "Deployment", "activator", `$patch: replace`),
			wantError: "$patch directive is only supported for the strategic merge patch type",
		},
		{
			name:      "root replacement rejected for json patch",
			resources: []unstructured.Unstructured{deploymentForPatchTest(t, "activator", "knative-serving", 1)},
			patch:     resourcePatchForTest(base.JSONPatchType, "apps/v1", "Deployment", "activator", `$patch: replace`),
			wantError: "$patch directive is only supported for the strategic merge patch type",
		},
		{
			name:      "nested directive rejected for merge patch",
			resources: []unstructured.Unstructured{deploymentForPatchTest(t, "activator", "knative-serving", 1)},
			patch: resourcePatchForTest(base.MergePatchType, "apps/v1", "Deployment", "activator", `
spec:
  template:
    spec:
      $patch: replace
`),
			wantError: "$patch directive is only supported for the strategic merge patch type",
		},
		{
			name:      "nested directive rejected in json patch value",
			resources: []unstructured.Unstructured{deploymentForPatchTest(t, "activator", "knative-serving", 1)},
			patch: resourcePatchForTest(base.JSONPatchType, "apps/v1", "Deployment", "activator", `
- op: add
  path: /metadata/annotations
  value:
    example.com/config:
      $patch: replace
`),
			wantError: "$patch directive is only supported for the strategic merge patch type",
		},
		{
			name:      "unsupported patch type",
			resources: []unstructured.Unstructured{deploymentForPatchTest(t, "activator", "knative-serving", 1)},
			patch:     resourcePatchForTest(base.PatchType("unknown"), "apps/v1", "Deployment", "activator", `{}`),
			wantError: "unsupported patch type",
		},
		{
			name:      "malformed JSON patch",
			resources: []unstructured.Unstructured{deploymentForPatchTest(t, "activator", "knative-serving", 1)},
			patch:     resourcePatchForTest(base.JSONPatchType, "apps/v1", "Deployment", "activator", `not: an-array`),
			wantError: "decode JSON patch",
		},
		{
			name:      "non-string root strategic directive",
			resources: []unstructured.Unstructured{deploymentForPatchTest(t, "activator", "knative-serving", 1)},
			patch:     resourcePatchForTest(base.StrategicMergePatchType, "apps/v1", "Deployment", "activator", `$patch: 1`),
			wantError: "root $patch directive must be a string",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manifest := manifestForPatchTest(t, test.resources...)
			test.patch.Target.Namespace = test.targetNamespace
			err := ApplyResourcePatches(&manifest, []base.ResourcePatch{test.patch})
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("ApplyResourcePatches() = %v, want error containing %q", err, test.wantError)
			}
		})
	}
}

func TestValidatePatchDirectivesReturnsDecodeError(t *testing.T) {
	for _, patchType := range []base.PatchType{base.JSONPatchType, base.MergePatchType} {
		t.Run(string(patchType), func(t *testing.T) {
			err := validatePatchDirectives(patchType, []byte(`{`))
			if err == nil || !strings.Contains(err.Error(), "decode patch document") {
				t.Fatalf("validatePatchDirectives() = %v, want decode patch document error", err)
			}
		})
	}
}

func TestResourcePatchRunsAfterBuiltInTransforms(t *testing.T) {
	replicas := int32(5)
	instance := &v1beta1.KnativeServing{
		ObjectMeta: metav1.ObjectMeta{Namespace: "knative-serving"},
		Spec: v1beta1.KnativeServingSpec{CommonSpec: base.CommonSpec{
			HighAvailability: &base.HighAvailability{Replicas: &replicas},
			Patches: []base.ResourcePatch{
				resourcePatchForTest(base.StrategicMergePatchType, "apps/v1", "Deployment", "controller", `
spec:
  replicas: null
`),
			},
		}},
	}
	manifest := manifestForPatchTest(t, deploymentForPatchTest(t, "controller", "upstream-namespace", 1))

	if err := Transform(context.Background(), &manifest, instance); err != nil {
		t.Fatalf("Transform() = %v", err)
	}
	patchStages := NewResourcePatchStages()
	if err := patchStages.Apply(context.Background(), &manifest, instance); err != nil {
		t.Fatalf("ResourcePatchStages.Apply() = %v", err)
	}
	got := manifest.Resources()[0]
	if got.GetNamespace() != "knative-serving" {
		t.Fatalf("namespace = %q, want knative-serving", got.GetNamespace())
	}
	if _, found, err := unstructured.NestedFieldNoCopy(got.Object, "spec", "replicas"); err != nil || found {
		t.Fatalf("spec.replicas found = %v, err = %v, want false, nil", found, err)
	}
}

func TestResourcePatchFailureMarksInstallFailed(t *testing.T) {
	instance := &v1beta1.KnativeServing{
		ObjectMeta: metav1.ObjectMeta{Namespace: "knative-serving"},
		Spec: v1beta1.KnativeServingSpec{CommonSpec: base.CommonSpec{
			Patches: []base.ResourcePatch{
				resourcePatchForTest(base.MergePatchType, "apps/v1", "Deployment", "missing", `{}`),
			},
		}},
	}
	manifest := manifestForPatchTest(t, deploymentForPatchTest(t, "controller", "upstream-namespace", 1))

	patchStages := NewResourcePatchStages()
	if err := patchStages.Apply(context.Background(), &manifest, instance); err == nil {
		t.Fatal("ResourcePatchStages.Apply() = nil, want patch target error")
	}
	apistest.CheckConditionFailed(&instance.Status, base.InstallSucceeded, t)
}

func TestApplyPatchesRejectsProtectedDelete(t *testing.T) {
	instance := &v1beta1.KnativeServing{
		Spec: v1beta1.KnativeServingSpec{CommonSpec: base.CommonSpec{
			Patches: []base.ResourcePatch{
				resourcePatchForTest(base.StrategicMergePatchType, "apps/v1", "Deployment", "webhook", `$patch: delete`),
			},
		}},
	}
	manifest := manifestForPatchTest(t, deploymentForPatchTest(t, "webhook", "knative-serving", 1))
	protectedWebhook := base.PatchTarget{APIVersion: "apps/v1", Kind: "Deployment", Name: "webhook"}

	err := NewResourcePatchStages(protectedWebhook).Apply(context.Background(), &manifest, instance)
	if err == nil || !strings.Contains(err.Error(), "deleting protected resource") {
		t.Fatalf("ResourcePatchStages.Apply() = %v, want protected resource error", err)
	}
}

func TestManifestivalPreservesExternallyManagedReplicas(t *testing.T) {
	client := mffake.New()
	manifest, err := mf.ManifestFrom(
		mf.Slice([]unstructured.Unstructured{deploymentForPatchTestWithoutReplicas("activator", "knative-serving")}),
		mf.UseClient(client),
	)
	if err != nil {
		t.Fatalf("ManifestFrom() = %v", err)
	}
	if err := manifest.Apply(); err != nil {
		t.Fatalf("initial Apply() = %v", err)
	}

	deploymentIdentity := deploymentForPatchTestWithoutReplicas("activator", "knative-serving")
	liveDeployment, err := client.Get(&deploymentIdentity)
	if err != nil {
		t.Fatalf("Get() = %v", err)
	}
	if err := unstructured.SetNestedField(liveDeployment.Object, int64(7), "spec", "replicas"); err != nil {
		t.Fatalf("SetNestedField() = %v", err)
	}
	if err := client.Update(liveDeployment); err != nil {
		t.Fatalf("Update() = %v", err)
	}

	if err := manifest.Apply(); err != nil {
		t.Fatalf("second Apply() = %v", err)
	}
	liveDeployment, err = client.Get(&deploymentIdentity)
	if err != nil {
		t.Fatalf("Get() after reconcile = %v", err)
	}
	if replicas, _, err := unstructured.NestedInt64(liveDeployment.Object, "spec", "replicas"); err != nil || replicas != 7 {
		t.Fatalf("live replicas = %d, %v, want 7, nil", replicas, err)
	}
}

func resourcePatchForTest(patchType base.PatchType, apiVersion, kind, name, content string) base.ResourcePatch {
	return base.ResourcePatch{
		Target: base.PatchTarget{APIVersion: apiVersion, Kind: kind, Name: name},
		Patch:  base.PatchSpec{Type: patchType, Content: content},
	}
}

func manifestForPatchTest(t *testing.T, resources ...unstructured.Unstructured) mf.Manifest {
	t.Helper()
	manifest, err := mf.ManifestFrom(mf.Slice(resources))
	if err != nil {
		t.Fatalf("ManifestFrom() = %v", err)
	}
	return manifest
}

func resourceForPatchTest(apiVersion, kind, name, namespace string) unstructured.Unstructured {
	return *NamespacedResource(apiVersion, kind, namespace, name)
}

func deploymentForPatchTest(t *testing.T, name, namespace string, replicas int64) unstructured.Unstructured {
	t.Helper()
	deployment := deploymentForPatchTestWithoutReplicas(name, namespace)
	if err := unstructured.SetNestedField(deployment.Object, replicas, "spec", "replicas"); err != nil {
		t.Fatalf("SetNestedField() = %v", err)
	}
	return deployment
}

func deploymentForPatchTestWithoutReplicas(name, namespace string) unstructured.Unstructured {
	return resourceForPatchTest("apps/v1", "Deployment", name, namespace)
}
