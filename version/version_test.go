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
	"fmt"
	util "knative.dev/operator/pkg/reconciler/common/testing"
	"path/filepath"
	"runtime"
	"strings"
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
	manifest, err := mf.NewManifest(filepath.Join(filepath.Dir(b)+"/..", "cmd/operator/kodata/knative-eventing/0.14.2"))
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

func TestManifestVersionEventingSame1(t *testing.T) {
	yaml := "upgrade-to-v%s.yaml"
	component := "eventing"
	version := "0.15.0"
	RELEASE_LINK := "https://github.com/knative/%s/releases/download/v%s/%s"
	file := yaml
	if strings.Contains(yaml, "%s") {
		file = fmt.Sprintf(yaml, version)
	}
	fmt.Println(file)
	fileLink := fmt.Sprintf(RELEASE_LINK, component, version, file)
	fmt.Println(fileLink)
	manifest, err := mf.NewManifest(fileLink)
	if err != nil {
		t.Fatal("Failed to load manifest", err)
	}
	util.AssertEqual(t, len(manifest.Resources()) == 0, false)

	yaml = "eventing-crds.yaml"
	file = yaml
	if strings.Contains(yaml, "%s") {
		file = fmt.Sprintf(yaml, version)
	}
	fmt.Println(file)
	fileLink = fmt.Sprintf(RELEASE_LINK, component, version, file)
	fmt.Println(fileLink)
	manifest1, err1 := mf.NewManifest(fileLink)
	if err1 != nil {
		t.Fatal("Failed to load manifest", err)
	}

	fmt.Println("old manifest")
	fmt.Println(manifest)
	manifest = manifest.Append(manifest1)

	fmt.Println("new manifest")
	fmt.Println(manifest)
	util.AssertEqual(t, len(manifest.Resources()), 100)

}
