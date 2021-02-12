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
	"testing"

	mf "github.com/manifestival/manifestival"
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
)

type TestExtension string

func (t TestExtension) Manifests() []mf.Manifest {
	manifest, err := mf.NewManifest("testdata/kodata/additional-manifests/additional-sa.yaml")
	if err != nil {
		return nil
	}
	return []mf.Manifest {manifest}
}

func (t TestExtension) Transformers(v1alpha1.KComponent) []mf.Transformer {
	if t == "fail" {
		return nil
	}
	return []mf.Transformer{mf.InjectNamespace(string(t))}
}

func (t TestExtension) Reconcile(context.Context, v1alpha1.KComponent) error {
	return nil
}
func (t TestExtension) Finalize(context.Context, v1alpha1.KComponent) error {
	return nil
}

func TestExtensions(t *testing.T) {
	tests := []struct {
		name     string
		platform Extension
		length   int
	}{{
		name:     "happy path",
		platform: TestExtension("happy"),
		length:   1,
	}, {
		name:     "sad path",
		platform: TestExtension("fail"),
		length:   0,
	}, {
		name:     "nil path",
		platform: nil,
		length:   0,
	}, {
		name:     "no path",
		platform: NoExtension(context.TODO()),
		length:   0,
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ext := test.platform
			if ext != nil {
				transformers := ext.Transformers(nil)
				if len(transformers) != test.length {
					t.Error("Unexpected result")
				}
				if ext.Reconcile(context.TODO(), nil) != nil {
					t.Error("Unexpected result")
				}
				if test.length == 1 {
					manifest := test.platform.Manifests()[0]
					if err := Transform(context.TODO(), &manifest, &v1alpha1.KnativeServing{}, transformers...); err != nil {
						t.Error("Unexpected result")
					}
					for _, r := range manifest.Resources() {
						if r.GetNamespace() != string(ext.(TestExtension)) {
							t.Error("Unexpected result")
						}
					}
				}
				if ext.Finalize(context.TODO(), nil) != nil {
					t.Error("Unexpected result")
				}
			}
		})
	}
}
