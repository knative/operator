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
	"os"
	"testing"

	util "knative.dev/operator/pkg/reconciler/common/testing"
)

func TestGetLatestKodataReleaseTag(t *testing.T) {
	os.Setenv("KO_DATA_PATH", "testdata/kodata")
	releaseTag, err := GetLatestKodataReleaseTag("knative-serving")
	util.AssertEqual(t, err, nil)
	util.AssertEqual(t, releaseTag, "0.15.0")
	os.Unsetenv("KO_DATA_PATH")
}
