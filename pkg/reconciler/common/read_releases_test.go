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
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	mf "github.com/manifestival/manifestival"
	util "knative.dev/operator/pkg/reconciler/common/testing"
)

func TestRetrieveManifestPath(t *testing.T) {
	koPath := "../../../cmd/operator/kodata"

	tests := []struct {
		component string
		version   string
		name      string
	}{{
		name:      "Valid Knative Serving Version",
		component: "knative-serving",
		version:   "0.14.0",
	}, {
		name:      "Valid Knative Eventing Version",
		component: "knative-eventing",
		version:   "0.14.2",
	}}

	os.Setenv(KoEnvKey, koPath)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manifestPath := RetrieveManifestPath(test.version, test.component)
			expected := fmt.Sprintf("%s/%s/%s", koPath, test.component, test.version)
			util.AssertEqual(t, manifestPath, expected)
			manifest, err := mf.NewManifest(manifestPath)
			util.AssertEqual(t, err, nil)
			util.AssertEqual(t, len(manifest.Resources()) > 0, true)
		})
	}

	invalidPathTests := []struct {
		component string
		version   string
		name      string
	}{{
		name:      "Invalid Knative Serving Version",
		component: "knative-serving",
		version:   "invalid-version",
	}, {
		name:      "Invalid Knative component name",
		component: "invalid-component",
		version:   "0.14.2",
	}}

	for _, test := range invalidPathTests {
		t.Run(test.component, func(t *testing.T) {
			manifestPath := RetrieveManifestPath(test.version, test.component)
			expected := fmt.Sprintf("%s/%s/%s", koPath, test.component, test.version)
			util.AssertEqual(t, manifestPath, expected)
			manifest, err := mf.NewManifest(manifestPath)
			util.AssertEqual(t, err != nil, true)
			util.AssertEqual(t, len(manifest.Resources()) == 0, true)
		})
	}

	os.Unsetenv(KoEnvKey)
}

func TestGetLatestRelease(t *testing.T) {
	_, b, _, _ := runtime.Caller(0)
	koPath := filepath.Dir(b) + "/../../../cmd/operator/kodata"

	tests := []struct {
		component string
		expected  string
	}{{
		component: "knative-serving",
		expected:  "0.14.0",
	}, {
		component: "knative-eventing",
		expected:  "0.14.2",
	}}

	os.Setenv(KoEnvKey, koPath)
	for _, test := range tests {
		t.Run(test.component, func(t *testing.T) {
			version := GetLatestRelease(test.component)
			util.AssertEqual(t, version, test.expected)
		})
	}
	os.Unsetenv(KoEnvKey)
}

func TestManifestVersionTheSame(t *testing.T) {
	_, b, _, _ := runtime.Caller(0)
	koPath := filepath.Dir(b) + "/../../../cmd/operator/kodata"

	tests := []struct {
		component string
		label     string
	}{{
		component: "knative-serving",
		label:     "serving.knative.dev/release",
	}, {
		component: "knative-eventing",
		label:     "eventing.knative.dev/release",
	}}

	os.Setenv(KoEnvKey, koPath)
	for _, test := range tests {
		t.Run(test.component, func(t *testing.T) {
			versionList := ListReleases(test.component)

			// Check all the available version under the directory of each Knative component
			for _, version := range versionList {
				manifest, err := mf.NewManifest(filepath.Join(koPath, test.component, version))
				util.AssertEqual(t, err, nil)
				expectedLabelValue := "v" + version
				for _, resource := range manifest.Filter(mf.ByLabel(test.label, "")).Resources() {
					label := resource.GetLabels()[test.label]
					util.AssertEqual(t, label, expectedLabelValue)
				}

				fmt.Println(len(manifest.Filter(mf.ByKind("Deployment")).Resources()))
				for _, resource := range manifest.Filter(mf.ByKind("Deployment")).Resources() {
					fmt.Println(resource.GetName())
				}
			}
		})
	}
	os.Unsetenv(KoEnvKey)
}
