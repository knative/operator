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

	mf "github.com/manifestival/manifestival"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	util "knative.dev/operator/pkg/reconciler/common/testing"
)

func TestManifestDiffGenerator(t *testing.T) {
	oldManifest, err := mf.NewManifest("testdata/manifest.yaml")
	if err != nil {
		t.Fatalf("Failed to create manifest: %v", err)
	}
	newManifest, err := mf.NewManifest("testdata/manifestNew.yaml")
	if err != nil {
		t.Fatalf("Failed to create manifest: %v", err)
	}
	listCreate, listPatch, listDelete := ManifestDiffGenerator(oldManifest, newManifest)
	util.AssertEqual(t, len(listCreate), 1)
	existIn(t, listCreate, newManifest)
	notExistIn(t, listCreate, oldManifest)

	util.AssertEqual(t, len(listDelete), 1)
	existIn(t, listDelete, oldManifest)
	notExistIn(t, listDelete, newManifest)

	util.AssertEqual(t, len(listPatch), 55)
	existIn(t, listPatch, oldManifest)
	existIn(t, listPatch, newManifest)
}

func existIn(t *testing.T, listResource []unstructured.Unstructured, manifest mf.Manifest) {
	for _, val := range listResource {
		found, resource := FindResourceByNSGroupKindName(val, manifest.Resources())
		util.AssertEqual(t, found, true)
		util.AssertDeepEqual(t, val, resource)
	}
}

func notExistIn(t *testing.T, listResource []unstructured.Unstructured, manifest mf.Manifest) {
	for _, val := range listResource {
		found, resource := FindResourceByNSGroupKindName(val, manifest.Resources())
		util.AssertEqual(t, found, false)
		util.AssertDeepEqual(t, resource, unstructured.Unstructured{})
	}
}
