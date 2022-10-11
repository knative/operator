/*
Copyright 2022 The Knative Authors

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

	"github.com/google/go-cmp/cmp"
	v1 "k8s.io/api/core/v1"
)

func TestProbeTransform(t *testing.T) {
	tests := []struct {
		name     string
		override *v1.Probe
		tgt      *v1.Probe
		want     *v1.Probe
	}{{
		name: "multiple overrides",
		override: &v1.Probe{
			ProbeHandler: v1.ProbeHandler{
				Exec: &v1.ExecAction{Command: []string{"/test"}},
			},
			FailureThreshold: 5,
			TimeoutSeconds:   4,
		},
		tgt: &v1.Probe{
			ProbeHandler: v1.ProbeHandler{
				Exec: &v1.ExecAction{Command: []string{"/test2"}},
			},
			FailureThreshold: 8,
			TimeoutSeconds:   9,
			SuccessThreshold: 10,
		},
		want: &v1.Probe{
			ProbeHandler: v1.ProbeHandler{
				Exec: &v1.ExecAction{Command: []string{"/test"}},
			},
			FailureThreshold: 5,
			TimeoutSeconds:   4,
			SuccessThreshold: 10,
		},
	}, {
		name: "tgt with defaults",
		override: &v1.Probe{
			ProbeHandler: v1.ProbeHandler{
				Exec: &v1.ExecAction{Command: []string{"/test3"}},
			},
			FailureThreshold: 5,
			TimeoutSeconds:   4,
		},
		tgt: &v1.Probe{},
		want: &v1.Probe{
			ProbeHandler: v1.ProbeHandler{
				Exec: &v1.ExecAction{Command: []string{"/test3"}},
			},
			FailureThreshold: 5,
			TimeoutSeconds:   4,
		},
	}, {
		name:     "all with defaults",
		override: &v1.Probe{},
		tgt:      &v1.Probe{},
		want:     &v1.Probe{},
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mergeProbe(test.override, test.tgt)
			if diff := cmp.Diff(*test.want, *test.tgt); diff != "" {
				t.Fatalf("Probe merge failed: %v", diff)
			}
		})
	}
}
