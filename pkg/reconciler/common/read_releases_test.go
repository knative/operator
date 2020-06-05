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
	koPath := "testdata/kodata"

	tests := []struct {
		component string
		version   string
		name      string
		expected  string
	}{{
		name:      "Valid Knative Serving Version",
		component: "knative-serving",
		version:   "0.14.0",
		expected:  koPath + "/knative-serving/0.14.0",
	}, {
		name:      "Valid Knative Eventing Version",
		component: "knative-eventing",
		version:   "0.14.2",
		expected:  koPath + "/knative-eventing/0.14.2",
	}}

	os.Setenv(KoEnvKey, koPath)
	defer os.Unsetenv(KoEnvKey)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manifestPath := RetrieveManifestPath(test.version, test.component)
			util.AssertEqual(t, manifestPath, test.expected)
			manifest, err := mf.NewManifest(manifestPath)
			util.AssertEqual(t, err, nil)
			util.AssertEqual(t, len(manifest.Resources()) > 0, true)
		})
	}

	invalidPathTests := []struct {
		component string
		version   string
		name      string
		expected  string
	}{{
		name:      "Invalid Knative Serving Version",
		component: "knative-serving",
		version:   "invalid-version",
		expected:  koPath + "/knative-serving/invalid-version",
	}, {
		name:      "Invalid Knative component name",
		component: "invalid-component",
		version:   "0.14.2",
		expected:  koPath + "/invalid-component/0.14.2",
	}}

	for _, test := range invalidPathTests {
		t.Run(test.component, func(t *testing.T) {
			manifestPath := RetrieveManifestPath(test.version, test.component)
			util.AssertEqual(t, manifestPath, test.expected)
			manifest, err := mf.NewManifest(manifestPath)
			util.AssertEqual(t, err != nil, true)
			util.AssertEqual(t, len(manifest.Resources()) == 0, true)
		})
	}
}

func TestGetLatestRelease(t *testing.T) {
	koPath := "testdata/kodata"

	tests := []struct {
		component string
		expected  string
	}{{
		component: "knative-serving",
		expected:  "0.15.0",
	}, {
		component: "knative-eventing",
		expected:  "0.15.0",
	}}

	os.Setenv(KoEnvKey, koPath)
	defer os.Unsetenv(KoEnvKey)
	for _, test := range tests {
		t.Run(test.component, func(t *testing.T) {
			version := GetLatestRelease(test.component)
			util.AssertEqual(t, version, test.expected)
		})
	}
}

func TestListReleases(t *testing.T) {
	koPath := "testdata/kodata"

	tests := []struct {
		component string
		expected  []string
	}{{
		component: "knative-serving",
		expected:  []string{"0.15.0", "0.14.0"},
	}, {
		component: "knative-eventing",
		expected:  []string{"0.15.0", "0.14.2"},
	}}

	os.Setenv(KoEnvKey, koPath)
	defer os.Unsetenv(KoEnvKey)
	for _, test := range tests {
		t.Run(test.component, func(t *testing.T) {
			version, err := ListReleases(test.component)
			util.AssertEqual(t, err, nil)
			util.AssertDeepEqual(t, version, test.expected)
		})
	}
}

func TestListReleases1(t *testing.T) {
	_, b, _, _ := runtime.Caller(0)
	koPath := filepath.Dir(b)
	fmt.Println(koPath)
}
