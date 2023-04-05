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

package security

import (
	"context"
	"fmt"
	"os"
	"testing"

	mf "github.com/manifestival/manifestival"
	"knative.dev/operator/pkg/apis/operator/base"
	servingv1beta1 "knative.dev/operator/pkg/apis/operator/v1beta1"
	"knative.dev/operator/pkg/reconciler/common"
	util "knative.dev/operator/pkg/reconciler/common/testing"
)

func TestAppendTargetSecurityGuard(t *testing.T) {
	os.Setenv(common.KoEnvKey, "testdata/kodata")
	defer os.Unsetenv(common.KoEnvKey)

	tests := []struct {
		name                 string
		instance             servingv1beta1.KnativeServing
		expectedSecurityPath string
		expectedErr          error
	}{{
		name: "Available target security guard",
		instance: servingv1beta1.KnativeServing{
			Spec: servingv1beta1.KnativeServingSpec{
				CommonSpec: base.CommonSpec{
					Version: "1.8.0",
				},
				Security: &servingv1beta1.SecurityConfigs{
					SecurityGuard: base.SecurityGuardConfiguration{
						Enabled: true,
					},
				},
			},
		},
		expectedSecurityPath: os.Getenv(common.KoEnvKey) + "/security-guard/0.5",
		expectedErr:          nil,
	}, {
		name: "Available target security guard with disabled option",
		instance: servingv1beta1.KnativeServing{
			Spec: servingv1beta1.KnativeServingSpec{
				CommonSpec: base.CommonSpec{
					Version: "1.8.0",
				},
				Security: &servingv1beta1.SecurityConfigs{
					SecurityGuard: base.SecurityGuardConfiguration{
						Enabled: false,
					},
				},
			},
		},
		expectedSecurityPath: "",
		expectedErr:          nil,
	}, {
		name: "Available target security guard with empty security option",
		instance: servingv1beta1.KnativeServing{
			Spec: servingv1beta1.KnativeServingSpec{
				CommonSpec: base.CommonSpec{
					Version: "1.8.0",
				},
			},
		},
		expectedSecurityPath: "",
		expectedErr:          nil,
	}, {
		name: "Unavailable target security guard",
		instance: servingv1beta1.KnativeServing{
			Spec: servingv1beta1.KnativeServingSpec{
				CommonSpec: base.CommonSpec{
					Version: "1.7.1",
				},
				Security: &servingv1beta1.SecurityConfigs{
					SecurityGuard: base.SecurityGuardConfiguration{
						Enabled: true,
					},
				},
			},
		},
		expectedErr: fmt.Errorf("The current version of Knative Serving is 1.7. You need to install the version 1.8 or above to support the security guard"),
	}, {
		name: "Get the latest target security guard when the directory latest is unavailable",
		instance: servingv1beta1.KnativeServing{
			Spec: servingv1beta1.KnativeServingSpec{
				CommonSpec: base.CommonSpec{
					Version: "latest",
				},
				Security: &servingv1beta1.SecurityConfigs{
					SecurityGuard: base.SecurityGuardConfiguration{
						Enabled: true,
					},
				},
			},
		},
		expectedSecurityPath: os.Getenv(common.KoEnvKey) + "/security-guard/0.5",
		expectedErr:          nil,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifest, _ := mf.ManifestFrom(mf.Slice{})
			err := AppendTargetSecurity(context.TODO(), &manifest, &tt.instance)
			if err != nil {
				if tt.expectedErr == nil {
					t.Errorf("Unexpcted Error %v", err)
					return
				}
				util.AssertEqual(t, err.Error(), tt.expectedErr.Error())
				util.AssertEqual(t, len(manifest.Resources()), 0)
			} else {
				util.AssertEqual(t, util.DeepMatchWithPath(manifest, tt.expectedSecurityPath), true)
			}
		})
	}
}

// TODO: This test verifies the number of transformers. It should be rewritten by better test.
func TestTransformers(t *testing.T) {
	tests := []struct {
		name     string
		instance servingv1beta1.KnativeServing
		expected int
	}{{
		name: "Available security guard",
		instance: servingv1beta1.KnativeServing{
			Spec: servingv1beta1.KnativeServingSpec{
				CommonSpec: base.CommonSpec{
					Version: "1.8.0",
				},
				Security: &servingv1beta1.SecurityConfigs{
					SecurityGuard: base.SecurityGuardConfiguration{
						Enabled: true,
					},
				},
			},
		},
		expected: 1,
	}, {
		name: "Available security guard with disabled option",
		instance: servingv1beta1.KnativeServing{
			Spec: servingv1beta1.KnativeServingSpec{
				CommonSpec: base.CommonSpec{
					Version: "1.8.0",
				},
				Security: &servingv1beta1.SecurityConfigs{
					SecurityGuard: base.SecurityGuardConfiguration{
						Enabled: false,
					},
				},
			},
		},
		expected: 0,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transformers := Transformers(context.TODO(), &tt.instance)
			util.AssertEqual(t, len(transformers), tt.expected)
		})
	}
}
