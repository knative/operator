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
	mf "github.com/manifestival/manifestival"
	policyv1 "k8s.io/api/policy/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"

	"knative.dev/operator/pkg/apis/operator/base"
	servingv1beta1 "knative.dev/operator/pkg/apis/operator/v1beta1"
)

func TestPodDisruptionBudgetsTransformV1beta1(t *testing.T) {
	tests := []struct {
		name            string
		testName        string
		overrides       []base.PodDisruptionBudgetOverride
		expMinAvailable *intstr.IntOrString
	}{{
		name:     "simple override",
		testName: "activator-pdb",
		overrides: []base.PodDisruptionBudgetOverride{
			{
				Name: "activator-pdb",
				PodDisruptionBudgetSpec: policyv1.PodDisruptionBudgetSpec{
					MinAvailable: &intstr.IntOrString{StrVal: "50%", Type: intstr.String},
				},
			},
		},
		expMinAvailable: &intstr.IntOrString{StrVal: "50%", Type: intstr.String},
	}, {
		name:            "no override",
		overrides:       nil,
		testName:        "activator-pdb",
		expMinAvailable: &intstr.IntOrString{StrVal: "80%", Type: intstr.String},
	}, {
		name:     "override with no MinVailable",
		testName: "activator-pdb",
		overrides: []base.PodDisruptionBudgetOverride{
			{
				Name: "activator-pdb",
				PodDisruptionBudgetSpec: policyv1.PodDisruptionBudgetSpec{
					MinAvailable: nil,
				},
			},
		},
		expMinAvailable: &intstr.IntOrString{StrVal: "80%", Type: intstr.String},
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manifest, err := mf.NewManifest("testdata/manifest.yaml")
			if err != nil {
				t.Fatalf("Failed to create manifest: %v", err)
			}

			ks := &servingv1beta1.KnativeServing{
				Spec: servingv1beta1.KnativeServingSpec{
					CommonSpec: base.CommonSpec{
						PodDisruptionBudgetOverride: test.overrides,
					},
				},
			}

			manifest, err = manifest.Transform(PodDisruptionBudgetsTransform(ks, log))
			if err != nil {
				t.Fatalf("Failed to transform manifest: %v", err)
			}

			if test.overrides == nil {
				for _, u := range manifest.Resources() {
					if u.GetKind() == "PodDisruptionBudget" && u.GetName() == test.testName {
						got := &policyv1beta1.PodDisruptionBudget{}
						if err := scheme.Scheme.Convert(&u, got, nil); err != nil {
							t.Fatalf("Failed to convert unstructured to PodDisruptionBudget: %v", err)
						}

						if diff := cmp.Diff(*got.Spec.MinAvailable, *test.expMinAvailable); diff != "" {
							t.Fatalf("Unexpected minAvailable: %v", diff)
						}
					}
				}
			}

			for _, override := range test.overrides {
				for _, u := range manifest.Resources() {
					if u.GetKind() == "PodDisruptionBudget" && u.GetName() == override.Name {
						got := &policyv1beta1.PodDisruptionBudget{}
						if err := scheme.Scheme.Convert(&u, got, nil); err != nil {
							t.Fatalf("Failed to convert unstructured to PodDisruptionBudget: %v", err)
						}

						if diff := cmp.Diff(*got.Spec.MinAvailable, *test.expMinAvailable); diff != "" {
							t.Fatalf("Unexpected minAvailable: %v", diff)
						}
					}
				}
			}
		})
	}
}

func TestPodDisruptionBudgetsTransformV1(t *testing.T) {
	tests := []struct {
		name            string
		testName        string
		overrides       []base.PodDisruptionBudgetOverride
		expMinAvailable *intstr.IntOrString
	}{{
		name:     "simple override",
		testName: "activator-pdb-1",
		overrides: []base.PodDisruptionBudgetOverride{
			{
				Name: "activator-pdb-1",
				PodDisruptionBudgetSpec: policyv1.PodDisruptionBudgetSpec{
					MinAvailable: &intstr.IntOrString{StrVal: "50%", Type: intstr.String},
				},
			},
		},
		expMinAvailable: &intstr.IntOrString{StrVal: "50%", Type: intstr.String},
	}, {
		name:            "no override",
		overrides:       nil,
		testName:        "activator-pdb-1",
		expMinAvailable: &intstr.IntOrString{StrVal: "80%", Type: intstr.String},
	}, {
		name:     "override with no MinVailable",
		testName: "activator-pdb-1",
		overrides: []base.PodDisruptionBudgetOverride{
			{
				Name: "activator-pdb-1",
				PodDisruptionBudgetSpec: policyv1.PodDisruptionBudgetSpec{
					MinAvailable: nil,
				},
			},
		},
		expMinAvailable: &intstr.IntOrString{StrVal: "80%", Type: intstr.String},
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manifest, err := mf.NewManifest("testdata/manifest.yaml")
			if err != nil {
				t.Fatalf("Failed to create manifest: %v", err)
			}

			ks := &servingv1beta1.KnativeServing{
				Spec: servingv1beta1.KnativeServingSpec{
					CommonSpec: base.CommonSpec{
						PodDisruptionBudgetOverride: test.overrides,
					},
				},
			}

			manifest, err = manifest.Transform(PodDisruptionBudgetsTransform(ks, log))
			if err != nil {
				t.Fatalf("Failed to transform manifest: %v", err)
			}

			if test.overrides == nil {
				for _, u := range manifest.Resources() {
					if u.GetKind() == "PodDisruptionBudget" && u.GetName() == test.testName {
						got := &policyv1.PodDisruptionBudget{}
						if err := scheme.Scheme.Convert(&u, got, nil); err != nil {
							t.Fatalf("Failed to convert unstructured to PodDisruptionBudget: %v", err)
						}

						if diff := cmp.Diff(*got.Spec.MinAvailable, *test.expMinAvailable); diff != "" {
							t.Fatalf("Unexpected minAvailable: %v", diff)
						}
					}
				}
			}

			for _, override := range test.overrides {
				for _, u := range manifest.Resources() {
					if u.GetKind() == "PodDisruptionBudget" && u.GetName() == override.Name {
						got := &policyv1.PodDisruptionBudget{}
						if err := scheme.Scheme.Convert(&u, got, nil); err != nil {
							t.Fatalf("Failed to convert unstructured to PodDisruptionBudget: %v", err)
						}

						if diff := cmp.Diff(*got.Spec.MinAvailable, *test.expMinAvailable); diff != "" {
							t.Fatalf("Unexpected minAvailable: %v", diff)
						}
					}
				}
			}
		})
	}
}
