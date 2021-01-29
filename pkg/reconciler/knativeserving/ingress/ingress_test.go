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
	"os"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	mf "github.com/manifestival/manifestival"
	servingv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
	"knative.dev/operator/pkg/reconciler/common"
	util "knative.dev/operator/pkg/reconciler/common/testing"
)

const numberIngressResource = 27

func TestGetIngress(t *testing.T) {
	os.Setenv(common.KoEnvKey, "testdata/kodata")
	defer os.Unsetenv(common.KoEnvKey)

	tests := []struct {
		name                 string
		targetVersion        string
		expected             bool
		expectedIngressesNum int
	}{{
		name:                 "Available ingresses",
		targetVersion:        "0.18",
		expected:             true,
		expectedIngressesNum: numberIngressResource,
	}, {
		name:                 "Unavailable ingresses",
		targetVersion:        "0.16",
		expected:             false,
		expectedIngressesNum: 0,
	}, {
		name:                 "Missing version",
		targetVersion:        "",
		expected:             true,
		expectedIngressesNum: 0,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifest, _ := mf.ManifestFrom(mf.Slice{})
			err := getIngress(tt.targetVersion, &manifest)
			util.AssertEqual(t, err == nil, tt.expected)
			util.AssertEqual(t, len(manifest.Resources()), tt.expectedIngressesNum)
		})
	}
}

func TestAppendInstalledIngresses(t *testing.T) {
	os.Setenv(common.KoEnvKey, "testdata/kodata")
	defer os.Unsetenv(common.KoEnvKey)

	tests := []struct {
		name                 string
		instance             servingv1alpha1.KnativeServing
		expected             bool
		expectedIngressesNum int
	}{{
		name: "Available installed ingresses",
		instance: servingv1alpha1.KnativeServing{
			Spec: servingv1alpha1.KnativeServingSpec{},
			Status: servingv1alpha1.KnativeServingStatus{
				Version: "0.18.1",
			},
		},
		expected:             true,
		expectedIngressesNum: numberIngressResource,
	}, {
		name: "Available installed ingresses for missing status.version",
		instance: servingv1alpha1.KnativeServing{
			Spec: servingv1alpha1.KnativeServingSpec{
				CommonSpec: servingv1alpha1.CommonSpec{
					Version: "0.18.1",
				},
			},
			Status: servingv1alpha1.KnativeServingStatus{},
		},
		expected:             true,
		expectedIngressesNum: numberIngressResource,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifest, _ := mf.ManifestFrom(mf.Slice{})
			err := AppendInstalledIngresses(context.TODO(), &manifest, &tt.instance)
			util.AssertEqual(t, err == nil, tt.expected)
			util.AssertEqual(t, len(manifest.Resources()), tt.expectedIngressesNum)
		})
	}
}

func TestAppendTargetIngresses(t *testing.T) {
	os.Setenv(common.KoEnvKey, "testdata/kodata")
	defer os.Unsetenv(common.KoEnvKey)

	tests := []struct {
		name                 string
		instance             servingv1alpha1.KnativeServing
		expected             bool
		expectedIngressesNum int
	}{{
		name: "Available installed ingresses",
		instance: servingv1alpha1.KnativeServing{
			Spec: servingv1alpha1.KnativeServingSpec{
				CommonSpec: servingv1alpha1.CommonSpec{
					Version: "0.18.1",
				},
			},
		},
		expected:             true,
		expectedIngressesNum: numberIngressResource,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifest, _ := mf.ManifestFrom(mf.Slice{})
			err := AppendTargetIngresses(context.TODO(), &manifest, &tt.instance)
			util.AssertEqual(t, err == nil, tt.expected)
			util.AssertEqual(t, len(manifest.Resources()), tt.expectedIngressesNum)
		})
	}
}

func TestIngressFilter(t *testing.T) {
	tests := []struct {
		name        string
		ingressName string
		label       map[string]string
		expected    bool
	}{{
		name:        "Available installed ingresses",
		ingressName: "istio",
		label: map[string]string{
			"networking.knative.dev/ingress-provider": "istio",
		},
		expected: true,
	}, {
		name:        "Missing ingress label",
		ingressName: "istio",
		label:       map[string]string{},
		expected:    true,
	}, {
		name:        "Wrong ingress label",
		ingressName: "istio",
		label: map[string]string{
			"networking.knative.dev/ingress-provider": "kourier",
		},
		expected: false,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := makeUnstructured(t, "test-resource", tt.label)
			result := ingressFilter(tt.ingressName)(u)
			util.AssertEqual(t, result, tt.expected)
		})
	}
}

func makeUnstructured(t *testing.T, name string, labels map[string]string) *unstructured.Unstructured {
	d := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Spec: appsv1.DeploymentSpec{},
	}
	result := &unstructured.Unstructured{}
	err := scheme.Scheme.Convert(d, result, nil)
	if err != nil {
		t.Fatalf("Could not create unstructured Deployment: %v, err: %v", d, err)
	}
	return result
}

func TestFilters(t *testing.T) {
	tests := []struct {
		name     string
		instance servingv1alpha1.KnativeServing
		label    map[string]string
		expected bool
	}{{
		name: "Available istio ingress",
		instance: servingv1alpha1.KnativeServing{
			Spec: servingv1alpha1.KnativeServingSpec{
				Ingress: &servingv1alpha1.IngressConfigs{
					Istio: servingv1alpha1.IstioIngressConfiguration{
						Enabled: true,
					},
				},
			},
		},
		label: map[string]string{
			"networking.knative.dev/ingress-provider": "istio",
		},
		expected: true,
	}, {
		name: "Available kourier ingress",
		label: map[string]string{
			"networking.knative.dev/ingress-provider": "kourier",
		},
		instance: servingv1alpha1.KnativeServing{
			Spec: servingv1alpha1.KnativeServingSpec{
				Ingress: &servingv1alpha1.IngressConfigs{
					Kourier: servingv1alpha1.KourierIngressConfiguration{
						Enabled: true,
					},
				},
			},
		},
		expected: true,
	}, {
		name: "Available contour ingress",
		label: map[string]string{
			"networking.knative.dev/ingress-provider": "contour",
		},
		instance: servingv1alpha1.KnativeServing{
			Spec: servingv1alpha1.KnativeServingSpec{
				Ingress: &servingv1alpha1.IngressConfigs{
					Contour: servingv1alpha1.ContourIngressConfiguration{
						Enabled: true,
					},
				},
			},
		},
		expected: true,
	}, {
		name: "Empty ingress for default istio",
		label: map[string]string{
			"networking.knative.dev/ingress-provider": "istio",
		},
		instance: servingv1alpha1.KnativeServing{
			Spec: servingv1alpha1.KnativeServingSpec{},
		},
		expected: true,
	}, {
		name: "Empty ingress for non default ingress",
		label: map[string]string{
			"networking.knative.dev/ingress-provider": "kourier",
		},
		instance: servingv1alpha1.KnativeServing{
			Spec: servingv1alpha1.KnativeServingSpec{},
		},
		expected: false,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := makeUnstructured(t, "test-resource", tt.label)
			result := Filters(&tt.instance)(u)
			util.AssertEqual(t, result, tt.expected)
		})
	}
}

func TestTransformers(t *testing.T) {
	tests := []struct {
		name     string
		instance servingv1alpha1.KnativeServing
		expected int
	}{{
		name: "Available istio ingress",
		instance: servingv1alpha1.KnativeServing{
			Spec: servingv1alpha1.KnativeServingSpec{
				Ingress: &servingv1alpha1.IngressConfigs{
					Istio: servingv1alpha1.IstioIngressConfiguration{
						Enabled: true,
					},
				},
			},
		},
		expected: 1,
	}, {
		name: "Available kourier ingress",
		instance: servingv1alpha1.KnativeServing{
			Spec: servingv1alpha1.KnativeServingSpec{
				Ingress: &servingv1alpha1.IngressConfigs{
					Kourier: servingv1alpha1.KourierIngressConfiguration{
						Enabled: true,
					},
				},
			},
		},
		expected: 1,
	}, {
		name: "Available contour ingress",
		instance: servingv1alpha1.KnativeServing{
			Spec: servingv1alpha1.KnativeServingSpec{
				Ingress: &servingv1alpha1.IngressConfigs{
					Contour: servingv1alpha1.ContourIngressConfiguration{
						Enabled: true,
					},
				},
			},
		},
		expected: 0,
	}, {
		name: "Empty ingress for default istio",
		instance: servingv1alpha1.KnativeServing{
			Spec: servingv1alpha1.KnativeServingSpec{},
		},
		expected: 1,
	}, {
		name: "All ingresses enabled",
		instance: servingv1alpha1.KnativeServing{
			Spec: servingv1alpha1.KnativeServingSpec{
				Ingress: &servingv1alpha1.IngressConfigs{
					Contour: servingv1alpha1.ContourIngressConfiguration{
						Enabled: true,
					},
					Kourier: servingv1alpha1.KourierIngressConfiguration{
						Enabled: true,
					},
					Istio: servingv1alpha1.IstioIngressConfiguration{
						Enabled: true,
					},
				},
			},
		},
		expected: 2,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transformers := Transformers(context.TODO(), &tt.instance)
			util.AssertEqual(t, len(transformers), tt.expected)
		})
	}
}
