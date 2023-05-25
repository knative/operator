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

package ingress

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

// TODO: This test verifies the number of transformers. It should be rewritten by better test.
func TestTransformers(t *testing.T) {
	tests := []struct {
		name     string
		instance servingv1beta1.KnativeServing
		expected int
	}{{
		name: "Available istio ingress",
		instance: servingv1beta1.KnativeServing{
			Spec: servingv1beta1.KnativeServingSpec{
				Ingress: &servingv1beta1.IngressConfigs{
					Istio: base.IstioIngressConfiguration{
						Enabled: true,
					},
				},
			},
		},
		expected: 1,
	}, {
		name: "Available kourier ingress",
		instance: servingv1beta1.KnativeServing{
			Spec: servingv1beta1.KnativeServingSpec{
				Ingress: &servingv1beta1.IngressConfigs{
					Kourier: base.KourierIngressConfiguration{
						Enabled: true,
					},
				},
			},
		},
		expected: 3,
	}, {
		name: "Available contour ingress",
		instance: servingv1beta1.KnativeServing{
			Spec: servingv1beta1.KnativeServingSpec{
				Ingress: &servingv1beta1.IngressConfigs{
					Contour: base.ContourIngressConfiguration{
						Enabled: true,
					},
				},
			},
		},
		expected: 0,
	}, {
		name: "Empty ingress for default istio",
		instance: servingv1beta1.KnativeServing{
			Spec: servingv1beta1.KnativeServingSpec{},
		},
		expected: 1,
	}, {
		name: "All ingresses enabled",
		instance: servingv1beta1.KnativeServing{
			Spec: servingv1beta1.KnativeServingSpec{
				Ingress: &servingv1beta1.IngressConfigs{
					Contour: base.ContourIngressConfiguration{
						Enabled: true,
					},
					Kourier: base.KourierIngressConfiguration{
						Enabled: true,
					},
					Istio: base.IstioIngressConfiguration{
						Enabled: true,
					},
				},
			},
		},
		expected: 4,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transformers := Transformers(context.TODO(), &tt.instance)
			util.AssertEqual(t, len(transformers), tt.expected)
		})
	}
}

func TestGetIngress(t *testing.T) {
	os.Setenv(common.KoEnvKey, "testdata/kodata")
	defer os.Unsetenv(common.KoEnvKey)

	tests := []struct {
		name                string
		ingressPath         string
		expectedIngressPath string
		expectedErr         error
	}{{
		name:                "Available ingresses",
		ingressPath:         "testdata/kodata/ingress/1.9/kourier",
		expectedErr:         nil,
		expectedIngressPath: os.Getenv(common.KoEnvKey) + "/ingress/1.9/kourier",
	}, {
		name:        "Unavailable ingresses",
		ingressPath: "testdata/kodata/ingress/0.16/istio",
		expectedErr: fmt.Errorf("stat testdata/kodata/ingress/0.16/istio: no such file or directory"),
	}, {
		name:                "Missing version",
		ingressPath:         "",
		expectedErr:         nil,
		expectedIngressPath: "",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifest, _ := mf.ManifestFrom(mf.Slice{})
			m, err := getIngress(tt.ingressPath)
			if err != nil {
				util.AssertEqual(t, err.Error(), tt.expectedErr.Error())
				util.AssertEqual(t, len(manifest.Resources()), 0)
			} else {
				manifest = manifest.Append(m)
				util.AssertEqual(t, err, tt.expectedErr)
				util.AssertEqual(t, util.DeepMatchWithPath(manifest, tt.expectedIngressPath), true)
			}
		})
	}
}

func TestGetIngressPath(t *testing.T) {
	os.Setenv(common.KoEnvKey, "testdata/kodata")
	defer os.Unsetenv(common.KoEnvKey)

	tests := []struct {
		name         string
		version      string
		ks           *servingv1beta1.KnativeServing
		expectedPath string
	}{{
		name:         "Available ingress path for istio",
		version:      "1.9",
		ks:           &servingv1beta1.KnativeServing{},
		expectedPath: os.Getenv(common.KoEnvKey) + "/ingress/1.9/istio",
	}, {
		name:    "Available ingress path for istio with empty spec",
		version: "1.9",
		ks: &servingv1beta1.KnativeServing{
			Spec: servingv1beta1.KnativeServingSpec{},
		},
		expectedPath: os.Getenv(common.KoEnvKey) + "/ingress/1.9/istio",
	}, {
		name:    "Available ingress path for istio with nil ingress",
		version: "1.8",
		ks: &servingv1beta1.KnativeServing{
			Spec: servingv1beta1.KnativeServingSpec{
				Ingress: nil,
			},
		},
		expectedPath: os.Getenv(common.KoEnvKey) + "/ingress/1.8/istio",
	}, {
		name:    "Available ingress path for kourier",
		version: "1.8",
		ks: &servingv1beta1.KnativeServing{
			Spec: servingv1beta1.KnativeServingSpec{
				Ingress: &servingv1beta1.IngressConfigs{
					Kourier: base.KourierIngressConfiguration{
						Enabled: true,
					},
				},
			},
		},
		expectedPath: os.Getenv(common.KoEnvKey) + "/ingress/1.8/kourier",
	}, {
		name:    "Available ingress path for contour",
		version: "1.7.5",
		ks: &servingv1beta1.KnativeServing{
			Spec: servingv1beta1.KnativeServingSpec{
				Ingress: &servingv1beta1.IngressConfigs{
					Contour: base.ContourIngressConfiguration{
						Enabled: true,
					},
				},
			},
		},
		expectedPath: os.Getenv(common.KoEnvKey) + "/ingress/1.7/contour",
	}, {
		name:    "Available ingress path for contour of the latest version",
		version: "latest",
		ks: &servingv1beta1.KnativeServing{
			Spec: servingv1beta1.KnativeServingSpec{
				Ingress: &servingv1beta1.IngressConfigs{
					Contour: base.ContourIngressConfiguration{
						Enabled: true,
					},
				},
			},
		},
		expectedPath: os.Getenv(common.KoEnvKey) + "/ingress/latest/contour",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := GetIngressPath(tt.version, tt.ks)
			util.AssertEqual(t, path, tt.expectedPath)
		})
	}
}

func TestAppendTargetIngress(t *testing.T) {
	os.Setenv(common.KoEnvKey, "testdata/kodata")
	defer os.Unsetenv(common.KoEnvKey)

	tests := []struct {
		name                string
		instance            servingv1beta1.KnativeServing
		expectedIngressPath string
		expectedErr         error
	}{{
		name: "Available target ingresses",
		instance: servingv1beta1.KnativeServing{
			Spec: servingv1beta1.KnativeServingSpec{
				CommonSpec: base.CommonSpec{
					Version: "1.9",
				},
			},
		},
		expectedIngressPath: os.Getenv(common.KoEnvKey) + "/ingress/1.9/istio",
		expectedErr:         nil,
	}, {
		name: "Available target ingresses with Istio specified",
		instance: servingv1beta1.KnativeServing{
			Spec: servingv1beta1.KnativeServingSpec{
				CommonSpec: base.CommonSpec{
					Version: "1.8",
				},
				Ingress: &servingv1beta1.IngressConfigs{
					Istio: base.IstioIngressConfiguration{
						Enabled: true,
					},
				},
			},
		},
		expectedIngressPath: os.Getenv(common.KoEnvKey) + "/ingress/1.8/istio",
		expectedErr:         nil,
	}, {
		name: "Available target ingresses with Kourier specified",
		instance: servingv1beta1.KnativeServing{
			Spec: servingv1beta1.KnativeServingSpec{
				CommonSpec: base.CommonSpec{
					Version: "1.8",
				},
				Ingress: &servingv1beta1.IngressConfigs{
					Kourier: base.KourierIngressConfiguration{
						Enabled: true,
					},
				},
			},
		},
		expectedIngressPath: os.Getenv(common.KoEnvKey) + "/ingress/1.8/kourier",
		expectedErr:         nil,
	}, {
		name: "Available target ingresses with Contour specified",
		instance: servingv1beta1.KnativeServing{
			Spec: servingv1beta1.KnativeServingSpec{
				CommonSpec: base.CommonSpec{
					Version: "1.8",
				},
				Ingress: &servingv1beta1.IngressConfigs{
					Contour: base.ContourIngressConfiguration{
						Enabled: true,
					},
				},
			},
		},
		expectedIngressPath: os.Getenv(common.KoEnvKey) + "/ingress/1.8/contour",
		expectedErr:         nil,
	}, {
		name: "Unavailable target ingresses",
		instance: servingv1beta1.KnativeServing{
			Spec: servingv1beta1.KnativeServingSpec{
				CommonSpec: base.CommonSpec{
					Version: "0.12",
				},
			},
		},
		expectedErr: fmt.Errorf("stat testdata/kodata/ingress/0.12/istio: no such file or directory"),
	}, {
		name: "Get the latest target ingresses when the directory latest is unavailable",
		instance: servingv1beta1.KnativeServing{
			Spec: servingv1beta1.KnativeServingSpec{
				CommonSpec: base.CommonSpec{
					Version: "latest",
				},
			},
		},
		expectedIngressPath: os.Getenv(common.KoEnvKey) + "/ingress/1.9/istio",
		expectedErr:         nil,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifest, _ := mf.ManifestFrom(mf.Slice{})
			err := AppendTargetIngress(context.TODO(), &manifest, &tt.instance)
			if err != nil {
				util.AssertEqual(t, err.Error(), tt.expectedErr.Error())
				util.AssertEqual(t, len(manifest.Resources()), 0)
			} else {
				util.AssertEqual(t, err, tt.expectedErr)
				util.AssertEqual(t, util.DeepMatchWithPath(manifest, tt.expectedIngressPath), true)
			}
		})
	}
}

func TestAppendInstalledIngresses(t *testing.T) {
	os.Setenv(common.KoEnvKey, "testdata/kodata")
	defer os.Unsetenv(common.KoEnvKey)

	tests := []struct {
		name                string
		instance            servingv1beta1.KnativeServing
		expectedIngressPath string
		expectedErr         error
	}{{
		name: "Available installed ingresses",
		instance: servingv1beta1.KnativeServing{
			Spec: servingv1beta1.KnativeServingSpec{},
			Status: servingv1beta1.KnativeServingStatus{
				Version: "1.8.0",
			},
		},
		expectedIngressPath: os.Getenv(common.KoEnvKey) + "/ingress/1.8/istio",
		expectedErr:         nil,
	}, {
		name: "Available installed ingresses for missing status.version",
		instance: servingv1beta1.KnativeServing{
			Spec: servingv1beta1.KnativeServingSpec{
				CommonSpec: base.CommonSpec{
					Version: "1.8.0",
				},
			},
			Status: servingv1beta1.KnativeServingStatus{},
		},
		expectedIngressPath: os.Getenv(common.KoEnvKey) + "/ingress/1.8/istio",
		expectedErr:         nil,
	}, {
		name: "Unavailable installed ingresses for the unavailable status.version",
		instance: servingv1beta1.KnativeServing{
			Spec: servingv1beta1.KnativeServingSpec{},
			Status: servingv1beta1.KnativeServingStatus{
				Version: "0.12.1",
			},
		},
		// We still return nil, even if the ingress is not available.
		expectedErr: nil,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifest, _ := mf.ManifestFrom(mf.Slice{})
			err := AppendInstalledIngresses(context.TODO(), &manifest, &tt.instance)
			if err != nil {
				util.AssertEqual(t, err.Error(), tt.expectedErr.Error())
				util.AssertEqual(t, len(manifest.Resources()), 0)
			} else {
				util.AssertEqual(t, err, tt.expectedErr)
				util.AssertEqual(t, util.DeepMatchWithPath(manifest, tt.expectedIngressPath), true)
			}
		})
	}
}
