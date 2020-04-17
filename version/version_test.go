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
package version

import (
	"path/filepath"
	"runtime"
	"testing"

	mf "github.com/manifestival/manifestival"
)

func TestManifestVersionServingSame(t *testing.T) {
	_, b, _, _ := runtime.Caller(0)
	manifest, err := mf.NewManifest(filepath.Join(filepath.Dir(b)+"/..", "cmd/operator/kodata/knative-serving/"))
	if err != nil {
		t.Fatal("Failed to load manifest", err)
	}

	// example: v0.10.1
	expectedLabelValue := "v" + ServingVersion
	label := "serving.knative.dev/release"

	for _, resource := range manifest.Filter(mf.ByLabel(label, "")).Resources() {
		v := resource.GetLabels()[label]
		if v != expectedLabelValue {
			t.Errorf("Version info in manifest and operator don't match. got: %v, want: %v. Resource GVK: %v, Resource name: %v", v, expectedLabelValue,
				resource.GroupVersionKind(), resource.GetName())
		}
	}
}

func TestManifestVersionEventingSame(t *testing.T) {
	_, b, _, _ := runtime.Caller(0)
	manifest, err := mf.NewManifest(filepath.Join(filepath.Dir(b)+"/..", "cmd/operator/kodata/knative-eventing/"))
	if err != nil {
		t.Fatal("Failed to load manifest", err)
	}

	// example: v0.10.1
	expectedLabelValue := "v" + EventingVersion
	label := "eventing.knative.dev/release"

	for _, resource := range manifest.Filter(mf.ByLabel(label, "")).Resources() {
		v := resource.GetLabels()[label]
		if v != expectedLabelValue {
			t.Errorf("Version info in manifest and operator don't match. got: %v, want: %v. Resource GVK: %v, Resource name: %v", v, expectedLabelValue,
				resource.GroupVersionKind(), resource.GetName())
		}
	}
}
