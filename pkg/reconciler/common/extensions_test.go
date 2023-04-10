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
	"knative.dev/operator/pkg/apis/operator/base"
	"knative.dev/operator/pkg/apis/operator/v1beta1"
)

type TestExtension string

func (t TestExtension) Manifests(base.KComponent) ([]mf.Manifest, error) {
	manifest, err := mf.NewManifest("testdata/kodata/additional-manifests/additional-sa.yaml")
	if err != nil {
		return nil, err
	}
	return []mf.Manifest{manifest}, nil
}

func (t TestExtension) Transformers(base.KComponent) []mf.Transformer {
	if t == "fail" {
		return nil
	}
	return []mf.Transformer{mf.InjectNamespace(string(t))}
}

func (t TestExtension) Reconcile(context.Context, base.KComponent) error {
	return nil
}
func (t TestExtension) Finalize(context.Context, base.KComponent) error {
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
		platform: NoExtension(context.TODO(), nil),
		length:   0,
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ext := test.platform
			if ext != nil {
				transformers := ext.Transformers(nil)
				if len(transformers) != test.length {
					t.Errorf("Unexpected transformers length. Expected %d, got %d", test.length, len(transformers))
				}
				if err := ext.Reconcile(context.TODO(), nil); err != nil {
					t.Errorf("Extensions reconcile failed. error: %v", err)
				}
				if test.length == 1 {
					manifests, err := test.platform.Manifests(nil)
					if err != nil {
						t.Errorf("Extensions manifests failed. error: %v", err)
					}
					if len(manifests) == 0 {
						t.Fatal("manifests is empty")
					}
					if err := Transform(context.TODO(), &manifests[0], &v1beta1.KnativeServing{}, transformers...); err != nil {
						t.Errorf("Transform failed. error: %v", err)
					}
					for _, r := range manifests[0].Resources() {
						if r.GetNamespace() != string(ext.(TestExtension)) {
							t.Logf("Expected namespace: %s, got: %s", string(ext.(TestExtension)), r.GetNamespace())
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
