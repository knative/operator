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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
	"knative.dev/operator/pkg/apis/operator/base"
	servingv1beta1 "knative.dev/operator/pkg/apis/operator/v1beta1"
	"knative.dev/operator/pkg/reconciler/common"
	util "knative.dev/operator/pkg/reconciler/common/testing"
)

func TestGetIngress(t *testing.T) {
	os.Setenv(common.KoEnvKey, "testdata/kodata")
	defer os.Unsetenv(common.KoEnvKey)

	tests := []struct {
		name                string
		version             string
		expectedIngressPath string
		expectedErr         error
	}{{
		name:                "Available ingresses",
		version:             "0.21.0",
		expectedErr:         nil,
		expectedIngressPath: os.Getenv(common.KoEnvKey) + "/ingress/0.21",
	}, {
		name:        "Unavailable ingresses",
		version:     "0.16.1",
		expectedErr: fmt.Errorf("stat testdata/kodata/ingress/0.16: no such file or directory"),
	}, {
		name:                "Missing version",
		version:             "",
		expectedErr:         nil,
		expectedIngressPath: "",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifest, _ := mf.ManifestFrom(mf.Slice{})
			m, err := getIngress(tt.version)
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
				Version: "0.21.0",
			},
		},
		expectedIngressPath: os.Getenv(common.KoEnvKey) + "/ingress/0.21",
		expectedErr:         nil,
	}, {
		name: "Available installed ingresses for missing status.version",
		instance: servingv1beta1.KnativeServing{
			Spec: servingv1beta1.KnativeServingSpec{
				CommonSpec: base.CommonSpec{
					Version: "0.21.0",
				},
			},
			Status: servingv1beta1.KnativeServingStatus{},
		},
		expectedIngressPath: os.Getenv(common.KoEnvKey) + "/ingress/0.21",
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

func TestAppendTargetIngresses(t *testing.T) {
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
					Version: "0.21.0",
				},
			},
		},
		expectedIngressPath: os.Getenv(common.KoEnvKey) + "/ingress/0.21",
		expectedErr:         nil,
	}, {
		name: "Unavailable target ingresses",
		instance: servingv1beta1.KnativeServing{
			Spec: servingv1beta1.KnativeServingSpec{
				CommonSpec: base.CommonSpec{
					Version: "0.12.1",
				},
			},
		},
		expectedErr: fmt.Errorf("stat testdata/kodata/ingress/0.12: no such file or directory"),
	}, {
		name: "Get the latest target ingresses when the directory latest is unavailable",
		instance: servingv1beta1.KnativeServing{
			Spec: servingv1beta1.KnativeServingSpec{
				CommonSpec: base.CommonSpec{
					Version: "latest",
				},
			},
		},
		expectedIngressPath: os.Getenv(common.KoEnvKey) + "/ingress/0.22",
		expectedErr:         nil,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifest, _ := mf.ManifestFrom(mf.Slice{})
			err := AppendTargetIngresses(context.TODO(), &manifest, &tt.instance)
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

func TestGetIngressWithFilters(t *testing.T) {
	os.Setenv(common.KoEnvKey, "testdata/kodata")
	defer os.Unsetenv(common.KoEnvKey)
	version := "0.21"
	tests := []struct {
		name                 string
		instance             servingv1beta1.KnativeServing
		expectedManifestPath string
		expected             bool
	}{{
		name: "Enabled Istio ingress for target manifests",
		instance: servingv1beta1.KnativeServing{
			Spec: servingv1beta1.KnativeServingSpec{
				CommonSpec: base.CommonSpec{
					Version: version,
				},
				Ingress: &servingv1beta1.IngressConfigs{
					Istio: base.IstioIngressConfiguration{
						Enabled: true,
					},
				},
			},
		},
		expected:             true,
		expectedManifestPath: os.Getenv(common.KoEnvKey) + "/ingress/" + version + "/net-istio.yaml",
	}, {
		name: "Enabled Contour ingress for target manifests",
		instance: servingv1beta1.KnativeServing{
			Spec: servingv1beta1.KnativeServingSpec{
				CommonSpec: base.CommonSpec{
					Version: version,
				},
				Ingress: &servingv1beta1.IngressConfigs{
					Contour: base.ContourIngressConfiguration{
						Enabled: true,
					},
				},
			},
		},
		expected:             true,
		expectedManifestPath: os.Getenv(common.KoEnvKey) + "/ingress/" + version + "/net-contour.yaml",
	}, {
		name: "Enabled Kourier ingress for target manifests",
		instance: servingv1beta1.KnativeServing{
			Spec: servingv1beta1.KnativeServingSpec{
				CommonSpec: base.CommonSpec{
					Version: version,
				},
				Ingress: &servingv1beta1.IngressConfigs{
					Kourier: base.KourierIngressConfiguration{
						Enabled: true,
					},
				},
			},
		},
		expected:             true,
		expectedManifestPath: os.Getenv(common.KoEnvKey) + "/ingress/" + version + "/kourier.yaml",
	}, {
		name: "Enabled Contour and Kourier ingress for target manifests",
		instance: servingv1beta1.KnativeServing{
			Spec: servingv1beta1.KnativeServingSpec{
				CommonSpec: base.CommonSpec{
					Version: version,
				},
				Ingress: &servingv1beta1.IngressConfigs{
					Kourier: base.KourierIngressConfiguration{
						Enabled: true,
					},
					Contour: base.ContourIngressConfiguration{
						Enabled: true,
					},
				},
			},
		},
		expected: true,
		expectedManifestPath: os.Getenv(common.KoEnvKey) + "/ingress/" + version + "/net-contour.yaml" + "," +
			os.Getenv(common.KoEnvKey) + "/ingress/" + version + "/kourier.yaml",
	}, {
		name: "Enabled Istio and Kourier ingress for target manifests",
		instance: servingv1beta1.KnativeServing{
			Spec: servingv1beta1.KnativeServingSpec{
				CommonSpec: base.CommonSpec{
					Version: version,
				},
				Ingress: &servingv1beta1.IngressConfigs{
					Kourier: base.KourierIngressConfiguration{
						Enabled: true,
					},
					Istio: base.IstioIngressConfiguration{
						Enabled: true,
					},
				},
			},
		},
		expected: true,
		expectedManifestPath: os.Getenv(common.KoEnvKey) + "/ingress/" + version + "/kourier.yaml" + "," +
			os.Getenv(common.KoEnvKey) + "/ingress/" + version + "/net-istio.yaml",
	}, {
		name: "Enabled Istio and Contour ingress for target manifests",
		instance: servingv1beta1.KnativeServing{
			Spec: servingv1beta1.KnativeServingSpec{
				CommonSpec: base.CommonSpec{
					Version: version,
				},
				Ingress: &servingv1beta1.IngressConfigs{
					Contour: base.ContourIngressConfiguration{
						Enabled: true,
					},
					Istio: base.IstioIngressConfiguration{
						Enabled: true,
					},
				},
			},
		},
		expected: true,
		expectedManifestPath: os.Getenv(common.KoEnvKey) + "/ingress/" + version + "/net-contour.yaml" + "," +
			os.Getenv(common.KoEnvKey) + "/ingress/" + version + "/net-istio.yaml",
	}, {
		name: "Enabled All ingresses for target manifests",
		instance: servingv1beta1.KnativeServing{
			Spec: servingv1beta1.KnativeServingSpec{
				CommonSpec: base.CommonSpec{
					Version: version,
				},
				Ingress: &servingv1beta1.IngressConfigs{
					Contour: base.ContourIngressConfiguration{
						Enabled: true,
					},
					Istio: base.IstioIngressConfiguration{
						Enabled: true,
					},
					Kourier: base.KourierIngressConfiguration{
						Enabled: true,
					},
				},
			},
		},
		expected:             true,
		expectedManifestPath: os.Getenv(common.KoEnvKey) + "/ingress/" + version,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targetIngressManifests, err := common.FetchManifest(tt.expectedManifestPath)
			util.AssertEqual(t, err, nil)
			manifest, _ := mf.ManifestFrom(mf.Slice{})
			m, err := getIngress(version)
			util.AssertEqual(t, err == nil, tt.expected)
			manifest = manifest.Append(m)
			manifest = manifest.Filter(Filters(&tt.instance))
			// The resources loaded with the enabled istio ingress returns exactly the same resources as we
			// expect from the ingress yaml file.
			// The manifest could have more resources than targetIngressManifests, because if the resource is not
			// labelled with the ingress provider, it will be kept. We can make sure all the resources in targetIngressManifests
			// exist in the manifest.
			util.AssertEqual(t, len(targetIngressManifests.Filter(mf.Not(mf.In(manifest))).Resources()), 0)
		})
	}
}

func TestIngressFilter(t *testing.T) {
	tests := []struct {
		name        string
		ingressName string
		label       string
		expected    bool
	}{{
		name:        "Available installed ingresses",
		ingressName: "istio",
		label:       "istio",
		expected:    true,
	}, {
		name:        "Missing ingress label",
		ingressName: "istio",
		label:       "",
		expected:    true,
	}, {
		name:        "Wrong ingress label",
		ingressName: "istio",
		label:       "kourier",
		expected:    false,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := makeIngressResource(t, "test-resource", "knative-serving", tt.label)
			result := ingressFilter(tt.ingressName)(u)
			util.AssertEqual(t, result, tt.expected)
		})
	}
}

// TestFilters checks if s certain resource with a network provider label will be correctly returned when passing
// the filters. If the resource is not labelled with the network provider label, it will be returned by default,
// regardless of the configuration of the filters.
func TestFilters(t *testing.T) {
	servicename := "test-service"
	namespace := "knative-serving"
	tests := []struct {
		name     string
		instance servingv1beta1.KnativeServing
		// This label is used to mark the tested resource to indicate which ingress it belongs to.
		// Empty label means no label for the resource.
		labels []string
		// The expected result indicates whether the resource is kept or not.
		// If it is true, the resource is kept after the filter.
		// If it is false, the resource is removed after the filter.
		expected []bool
	}{{
		name: "Enabled Istio ingress for all resources",
		instance: servingv1beta1.KnativeServing{
			Spec: servingv1beta1.KnativeServingSpec{
				Ingress: &servingv1beta1.IngressConfigs{
					Istio: base.IstioIngressConfiguration{
						Enabled: true,
					},
				},
			},
		},
		labels:   []string{"istio", "contour", "kourier", ""},
		expected: []bool{true, false, false, true},
	}, {
		name: "Default ingress for all resources",
		instance: servingv1beta1.KnativeServing{
			Spec: servingv1beta1.KnativeServingSpec{},
		},
		labels:   []string{"istio", "contour", "kourier", ""},
		expected: []bool{true, false, false, true},
	}, {
		name: "Enabled kourier ingress for all resources",
		instance: servingv1beta1.KnativeServing{
			Spec: servingv1beta1.KnativeServingSpec{
				Ingress: &servingv1beta1.IngressConfigs{
					Kourier: base.KourierIngressConfiguration{
						Enabled: true,
					},
				},
			},
		},
		labels:   []string{"istio", "contour", "kourier", ""},
		expected: []bool{false, false, true, true},
	}, {
		name: "Enabled Contour ingress for all resources",
		instance: servingv1beta1.KnativeServing{
			Spec: servingv1beta1.KnativeServingSpec{
				Ingress: &servingv1beta1.IngressConfigs{
					Contour: base.ContourIngressConfiguration{
						Enabled: true,
					},
				},
			},
		},
		labels:   []string{"istio", "contour", "kourier", ""},
		expected: []bool{false, true, false, true},
	}, {
		name: "Enabled Contour and Istio ingress for all resources",
		instance: servingv1beta1.KnativeServing{
			Spec: servingv1beta1.KnativeServingSpec{
				Ingress: &servingv1beta1.IngressConfigs{
					Contour: base.ContourIngressConfiguration{
						Enabled: true,
					},
					Istio: base.IstioIngressConfiguration{
						Enabled: true,
					},
				},
			},
		},
		labels:   []string{"istio", "contour", "kourier", ""},
		expected: []bool{true, true, false, true},
	}, {
		name: "Enabled Kourier and Istio ingress for all resources",
		instance: servingv1beta1.KnativeServing{
			Spec: servingv1beta1.KnativeServingSpec{
				Ingress: &servingv1beta1.IngressConfigs{
					Kourier: base.KourierIngressConfiguration{
						Enabled: true,
					},
					Istio: base.IstioIngressConfiguration{
						Enabled: true,
					},
				},
			},
		},
		labels:   []string{"istio", "contour", "kourier", ""},
		expected: []bool{true, false, true, true},
	}, {
		name: "Enabled Kourier and Contour ingress for all resources",
		instance: servingv1beta1.KnativeServing{
			Spec: servingv1beta1.KnativeServingSpec{
				Ingress: &servingv1beta1.IngressConfigs{
					Kourier: base.KourierIngressConfiguration{
						Enabled: true,
					},
					Contour: base.ContourIngressConfiguration{
						Enabled: true,
					},
				},
			},
		},
		labels:   []string{"istio", "contour", "kourier", ""},
		expected: []bool{false, true, true, true},
	}, {
		name: "Enabled All ingress for all resources",
		instance: servingv1beta1.KnativeServing{
			Spec: servingv1beta1.KnativeServingSpec{
				Ingress: &servingv1beta1.IngressConfigs{
					Istio: base.IstioIngressConfiguration{
						Enabled: true,
					},
					Kourier: base.KourierIngressConfiguration{
						Enabled: true,
					},
					Contour: base.ContourIngressConfiguration{
						Enabled: true,
					},
				},
			},
		},
		labels:   []string{"istio", "contour", "kourier", ""},
		expected: []bool{true, true, true, true},
	}, {
		name: "Disabled All ingress",
		instance: servingv1beta1.KnativeServing{
			Spec: servingv1beta1.KnativeServingSpec{
				Ingress: &servingv1beta1.IngressConfigs{
					Istio: base.IstioIngressConfiguration{
						Enabled: false,
					},
					Kourier: base.KourierIngressConfiguration{
						Enabled: false,
					},
					Contour: base.ContourIngressConfiguration{
						Enabled: false,
					},
				},
			},
		},
		labels:   []string{"istio", "contour", "kourier", ""},
		expected: []bool{false, false, false, true},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i, label := range tt.labels {
				ingressResource := makeIngressResource(t, servicename, namespace, label)
				result := Filters(&tt.instance)(ingressResource)
				util.AssertEqual(t, result, tt.expected[i])
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

func makeIngressResource(t *testing.T, name, ns, ingressLabel string) *unstructured.Unstructured {
	labels := map[string]string{}
	if ingressLabel != "" {
		labels = map[string]string{
			"networking.knative.dev/ingress-provider": ingressLabel,
		}
	}
	service := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			Labels:    labels,
		},
	}
	result := &unstructured.Unstructured{}
	err := scheme.Scheme.Convert(service, result, nil)
	if err != nil {
		t.Fatalf("Could not create unstructured Service: %v, err: %v", service, err)
	}

	return result
}
