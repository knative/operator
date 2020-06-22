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
	"errors"
	"testing"

	mf "github.com/manifestival/manifestival"
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
	util "knative.dev/operator/pkg/reconciler/common/testing"
)

type TestExtension string

func (t TestExtension) Transformers(v1alpha1.KComponent) ([]mf.Transformer, error) {
	if t == "fail" {
		return nil, errors.New(string(t))
	}
	return []mf.Transformer{mf.InjectNamespace(string(t))}, nil
}

func (t TestExtension) Reconcile(context.Context, v1alpha1.KComponent) error {
	return nil
}
func (t TestExtension) Finalize(context.Context, v1alpha1.KComponent) error {
	return nil
}

func TestExtensions(t *testing.T) {
	tests := []struct {
		name      string
		platform  Extension
		wantError bool
	}{{
		name:      "happy path",
		platform:  TestExtension("happy"),
		wantError: false,
	}, {
		name:      "sad path",
		platform:  TestExtension("fail"),
		wantError: true,
	}, {
		name:      "no path",
		platform:  nil,
		wantError: false,
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := WithPlatform(context.Background(), test.platform)
			ext := GetPlatform(ctx)
			util.AssertEqual(t, ext, test.platform)
			if ext != nil {
				transformers, err := ext.Transformers(nil)
				if !test.wantError {
					util.AssertEqual(t, err, nil)
					util.AssertEqual(t, len(transformers), 1)
				}
			}
		})
	}
}
