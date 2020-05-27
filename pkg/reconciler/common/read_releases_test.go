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

	util "knative.dev/operator/pkg/reconciler/common/testing"
)

func TestGetLatestOnlinePatchRelease(t *testing.T) {
	// The first release of 0.13.x is 0.13.0
	releaseTag, err := GetLatestOnlinePatchRelease("knative", "serving", "0.13.0")
	util.AssertEqual(t, err, nil)
	util.AssertEqual(t, releaseTag != "", true)
	// The latest patch release for 0.13.x is 0.13.3. We should get this version number.
	util.AssertEqual(t, releaseTag, "0.13.3")
}
