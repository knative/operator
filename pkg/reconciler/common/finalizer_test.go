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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/operator/pkg/apis/operator/base"
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
)

func TestFinalizerRemovalPatch(t *testing.T) {
	tests := []struct {
		name string
		in   base.KComponent
		want []byte
	}{{
		name: "other finalizer, do nothing",
		in: &v1alpha1.KnativeServing{
			ObjectMeta: metav1.ObjectMeta{
				ResourceVersion: "testVersion",
				Finalizers:      []string{"another-finalizer"},
			},
		},
	}, {
		name: "no finalizer, do nothing",
		in: &v1alpha1.KnativeServing{
			ObjectMeta: metav1.ObjectMeta{
				ResourceVersion: "testVersion",
			},
		},
	}, {
		name: "remove finalizer",
		in: &v1alpha1.KnativeServing{
			ObjectMeta: metav1.ObjectMeta{
				ResourceVersion: "testVersion",
				Finalizers:      []string{"test-finalizer"},
			},
		},
		want: []byte(`{"metadata":{"finalizers":[],"resourceVersion":"testVersion"}}`),
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			patch, err := FinalizerRemovalPatch(test.in, "test-finalizer")
			if err != nil {
				t.Fatalf("Failed to generate patch: %v", err)
			}

			patchStr := string(patch)
			wantStr := string(test.want)
			if patchStr != wantStr {
				t.Fatalf("patch = %s, want %s", patchStr, wantStr)
			}
		})
	}
}
