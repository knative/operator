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

	"github.com/google/go-cmp/cmp"
	mf "github.com/manifestival/manifestival"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/kubernetes/scheme"
	servingv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
)

var testResources = corev1.ResourceRequirements{
	Limits: corev1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse("100m"),
		corev1.ResourceMemory: resource.MustParse("128Mi"),
	},
	Requests: corev1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse("30m"),
		corev1.ResourceMemory: resource.MustParse("20Mi"),
	},
}

// default resources defined in testdata/manifest.yaml.
var controllerDefaultResources = corev1.ResourceRequirements{
	Limits: corev1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse("1000m"),
		corev1.ResourceMemory: resource.MustParse("1000Mi"),
	},
	Requests: corev1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse("100m"),
		corev1.ResourceMemory: resource.MustParse("100Mi"),
	},
}

type expDeployments struct {
	expLabels              map[string]string
	expTemplateLabels      map[string]string
	expAnnotations         map[string]string
	expTemplateAnnotations map[string]string
	expReplicas            int32
	expContainers          map[string]corev1.ResourceRequirements
}

func TestDeploymentsTransform(t *testing.T) {
	tests := []struct {
		name          string
		override      []servingv1alpha1.DeploymentOverride
		expDeployment map[string]expDeployments
	}{{
		name:     "no override",
		override: nil,
		expDeployment: map[string]expDeployments{"controller": {
			expLabels:              map[string]string{"serving.knative.dev/release": "v0.13.0"},
			expTemplateLabels:      map[string]string{"serving.knative.dev/release": "v0.13.0", "app": "controller"},
			expAnnotations:         nil,
			expTemplateAnnotations: map[string]string{"cluster-autoscaler.kubernetes.io/safe-to-evict": "true"},
			expReplicas:            0,
			expContainers:          map[string]corev1.ResourceRequirements{"controller": controllerDefaultResources},
		}},
	}, {
		name: "simple override",
		override: []servingv1alpha1.DeploymentOverride{
			{
				Name:        "controller",
				Labels:      map[string]string{"a": "b"},
				Annotations: map[string]string{"c": "d"},
				Replicas:    5,
				Containers: []servingv1alpha1.ContainerOverride{{
					Name:                 "controller",
					ResourceRequirements: testResources,
				}},
			},
		},
		expDeployment: map[string]expDeployments{"controller": {
			expLabels:              map[string]string{"serving.knative.dev/release": "v0.13.0", "a": "b"},
			expTemplateLabels:      map[string]string{"serving.knative.dev/release": "v0.13.0", "app": "controller", "a": "b"},
			expAnnotations:         map[string]string{"c": "d"},
			expTemplateAnnotations: map[string]string{"cluster-autoscaler.kubernetes.io/safe-to-evict": "true", "c": "d"},
			expReplicas:            5,
			expContainers:          map[string]corev1.ResourceRequirements{"controller": testResources},
		}},
	}, {
		name: "multiple override",
		override: []servingv1alpha1.DeploymentOverride{
			{
				Name:        "controller",
				Labels:      map[string]string{"a": "b"},
				Annotations: map[string]string{"c": "d"},
				Replicas:    5,
				Containers: []servingv1alpha1.ContainerOverride{{
					Name:                 "controller",
					ResourceRequirements: testResources,
				}},
			},
			{
				Name:        "webhook",
				Labels:      map[string]string{"e": "f"},
				Annotations: map[string]string{"g": "h"},
				Replicas:    4,
				Containers: []servingv1alpha1.ContainerOverride{{
					Name:                 "webhook",
					ResourceRequirements: testResources,
				}},
			},
		},
		expDeployment: map[string]expDeployments{
			"controller": {
				expLabels:              map[string]string{"serving.knative.dev/release": "v0.13.0", "a": "b"},
				expTemplateLabels:      map[string]string{"serving.knative.dev/release": "v0.13.0", "app": "controller", "a": "b"},
				expAnnotations:         map[string]string{"c": "d"},
				expTemplateAnnotations: map[string]string{"cluster-autoscaler.kubernetes.io/safe-to-evict": "true", "c": "d"},
				expReplicas:            5,
				expContainers:          map[string]corev1.ResourceRequirements{"controller": testResources},
			},
			"webhook": {
				expLabels:              map[string]string{"serving.knative.dev/release": "v0.13.0", "e": "f"},
				expTemplateLabels:      map[string]string{"serving.knative.dev/release": "v0.13.0", "app": "webhook", "role": "webhook", "e": "f"},
				expAnnotations:         map[string]string{"g": "h"},
				expTemplateAnnotations: map[string]string{"cluster-autoscaler.kubernetes.io/safe-to-evict": "false", "g": "h"},
				expReplicas:            4,
				expContainers:          map[string]corev1.ResourceRequirements{"webhook": testResources},
			},
		}}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manifest, err := mf.NewManifest("testdata/manifest.yaml")
			if err != nil {
				t.Fatalf("Failed to create manifest: %v", err)
			}

			ks := &servingv1alpha1.KnativeServing{
				Spec: servingv1alpha1.KnativeServingSpec{
					CommonSpec: servingv1alpha1.CommonSpec{
						DeploymentOverride: test.override,
					},
				},
			}

			manifest, err = manifest.Transform(DeploymentsTransform(ks, log))
			if err != nil {
				t.Fatalf("Failed to transform manifest: %v", err)
			}

			for expName, d := range test.expDeployment {
				for _, u := range manifest.Resources() {
					if u.GetKind() == "Deployment" && u.GetName() == expName {
						got := &appsv1.Deployment{}
						if err := scheme.Scheme.Convert(&u, got, nil); err != nil {
							t.Fatalf("Failed to convert unstructured to deployment: %v", err)
						}

						replicas := int32(0)
						if got.Spec.Replicas != nil {
							replicas = *got.Spec.Replicas
						}
						if diff := cmp.Diff(replicas, d.expReplicas); diff != "" {
							t.Fatalf("Unexpected replicas: %v", diff)
						}

						if diff := cmp.Diff(got.GetLabels(), d.expLabels); diff != "" {
							t.Fatalf("Unexpected labels: %v", diff)
						}
						if diff := cmp.Diff(got.Spec.Template.GetLabels(), d.expTemplateLabels); diff != "" {
							t.Fatalf("Unexpected labels in pod template: %v", diff)
						}

						if diff := cmp.Diff(got.GetAnnotations(), d.expAnnotations); diff != "" {
							t.Fatalf("Unexpected annotations: %v", diff)
						}
						if diff := cmp.Diff(got.Spec.Template.GetAnnotations(), d.expTemplateAnnotations); diff != "" {
							t.Fatalf("Unexpected annotations in pod template: %v", diff)
						}

						for _, c := range got.Spec.Template.Spec.Containers {
							resource := d.expContainers[c.Name]
							if diff := cmp.Diff(resource.Limits, c.Resources.Limits); diff != "" {
								t.Fatalf("Unexpected limits: %v", diff)
							}
							if diff := cmp.Diff(resource.Requests, c.Resources.Requests); diff != "" {
								t.Fatalf("Unexpected requests: %v", diff)
							}
						}
					}
				}
			}
		})
	}
}
