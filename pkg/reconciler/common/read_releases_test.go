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
	"context"
	"os"
	"testing"

	mf "github.com/manifestival/manifestival"

	util "knative.dev/operator/pkg/reconciler/common/testing"
)

func TestRetrieveManifest(t *testing.T) {
	tests := []struct {
		component string
		version   string
		label     string
	}{{
		component: "knative-serving",
		version:   "0.14.0",
		label:     "serving.knative.dev/release",
	}, {
		component: "knative-eventing",
		version:   "0.14.2",
		label:     "eventing.knative.dev/release",
	}}

	os.Setenv("KO_DATA_PATH", "../../../cmd/operator/kodata")
	for _, test := range tests {
		t.Run(test.component, func(t *testing.T) {
			manifest, err := RetrieveManifest(context.Background(), test.version, test.component)
			util.AssertEqual(t, err, nil)
			util.AssertEqual(t, len(manifest.Resources()) != 0, true)

			expectedLabelValue := "v" + test.version

			for _, resource := range manifest.Filter(mf.ByLabel(test.label, "")).Resources() {
				v := resource.GetLabels()[test.label]
				if v != expectedLabelValue {
					t.Errorf("Version info in manifest and operator don't match. got: %v, want: %v. Resource GVK: %v, Resource name: %v", v, expectedLabelValue,
						resource.GroupVersionKind(), resource.GetName())
				}
			}
		})
	}
	os.Unsetenv("KO_DATA_PATH")
}
