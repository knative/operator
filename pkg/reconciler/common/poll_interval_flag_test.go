/*
Copyright 2025 The Knative Authors

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
	"time"
)

func TestRemoteDeploymentsPollIntervalValue(t *testing.T) {
	cases := []struct {
		name string
		flag time.Duration
		want time.Duration
	}{
		{"default", defaultRemoteDeploymentsPollInterval, defaultRemoteDeploymentsPollInterval},
		{"valid override", 30 * time.Second, 30 * time.Second},
		{"below threshold clamps to default", 500 * time.Millisecond, defaultRemoteDeploymentsPollInterval},
		{"zero clamps to default", 0, defaultRemoteDeploymentsPollInterval},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			prev := remoteDeploymentsPollIntervalFlag
			remoteDeploymentsPollIntervalFlag = tc.flag
			t.Cleanup(func() { remoteDeploymentsPollIntervalFlag = prev })

			if got := RemoteDeploymentsPollIntervalValue(); got != tc.want {
				t.Errorf("RemoteDeploymentsPollIntervalValue() = %v, want %v", got, tc.want)
			}
		})
	}
}
