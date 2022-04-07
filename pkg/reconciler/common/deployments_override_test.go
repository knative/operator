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
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	mf "github.com/manifestival/manifestival"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"knative.dev/operator/pkg/apis/operator/base"
	servingv1beta1 "knative.dev/operator/pkg/apis/operator/v1beta1"
	util "knative.dev/operator/pkg/reconciler/common/testing"
	"knative.dev/operator/test"
)

type expDeployments struct {
	expLabels              map[string]string
	expTemplateLabels      map[string]string
	expAnnotations         map[string]string
	expTemplateAnnotations map[string]string
	expReplicas            int32
	expNodeSelector        map[string]string
	expTolerations         []corev1.Toleration
	expAffinity            *corev1.Affinity
}

func TestDeploymentsTransform(t *testing.T) {
	var four int32 = 4
	var five int32 = 5
	tests := []struct {
		name           string
		override       []base.DeploymentOverride
		globalReplicas int32
		expDeployment  map[string]expDeployments
	}{{
		name:     "no override",
		override: nil,
		expDeployment: map[string]expDeployments{"controller": {
			expLabels:              map[string]string{"serving.knative.dev/release": "v0.13.0"},
			expTemplateLabels:      map[string]string{"serving.knative.dev/release": "v0.13.0", "app": "controller"},
			expAnnotations:         nil,
			expTemplateAnnotations: map[string]string{"cluster-autoscaler.kubernetes.io/safe-to-evict": "true"},
			expReplicas:            0,
			expNodeSelector:        nil,
			expTolerations:         nil,
			expAffinity:            nil,
		}},
	}, {
		name: "simple override",
		override: []base.DeploymentOverride{
			{
				Name:         "controller",
				Labels:       map[string]string{"a": "b"},
				Annotations:  map[string]string{"c": "d"},
				Replicas:     &five,
				NodeSelector: map[string]string{"env": "prod"},
				Tolerations: []corev1.Toleration{{
					Key:      corev1.TaintNodeNotReady,
					Operator: corev1.TolerationOpExists,
					Effect:   corev1.TaintEffectNoSchedule,
				}},
				Affinity: &corev1.Affinity{
					NodeAffinity:    &corev1.NodeAffinity{},
					PodAffinity:     &corev1.PodAffinity{},
					PodAntiAffinity: &corev1.PodAntiAffinity{},
				},
			},
		},
		globalReplicas: 10,
		expDeployment: map[string]expDeployments{"controller": {
			expLabels:              map[string]string{"serving.knative.dev/release": "v0.13.0", "a": "b"},
			expTemplateLabels:      map[string]string{"serving.knative.dev/release": "v0.13.0", "app": "controller", "a": "b"},
			expAnnotations:         map[string]string{"c": "d"},
			expTemplateAnnotations: map[string]string{"cluster-autoscaler.kubernetes.io/safe-to-evict": "true", "c": "d"},
			expReplicas:            5,
			expNodeSelector:        map[string]string{"env": "prod"},
			expTolerations: []corev1.Toleration{{
				Key:      corev1.TaintNodeNotReady,
				Operator: corev1.TolerationOpExists,
				Effect:   corev1.TaintEffectNoSchedule,
			}},
			expAffinity: &corev1.Affinity{
				NodeAffinity:    &corev1.NodeAffinity{},
				PodAffinity:     &corev1.PodAffinity{},
				PodAntiAffinity: &corev1.PodAntiAffinity{},
			},
		}},
	}, {
		name: "no replicas in deploymentoverride, use global replicas",
		override: []base.DeploymentOverride{
			{Name: "controller"},
		},
		globalReplicas: 10,
		expDeployment: map[string]expDeployments{"controller": {
			expLabels:              map[string]string{"serving.knative.dev/release": "v0.13.0"},
			expTemplateLabels:      map[string]string{"serving.knative.dev/release": "v0.13.0", "app": "controller"},
			expTemplateAnnotations: map[string]string{"cluster-autoscaler.kubernetes.io/safe-to-evict": "true"},
			expReplicas:            10,
		}},
	}, {
		name: "neither replicas in deploymentoverride nor global replicas",
		override: []base.DeploymentOverride{
			{Name: "controller"},
		},
		expDeployment: map[string]expDeployments{"controller": {
			expLabels:              map[string]string{"serving.knative.dev/release": "v0.13.0"},
			expTemplateLabels:      map[string]string{"serving.knative.dev/release": "v0.13.0", "app": "controller"},
			expTemplateAnnotations: map[string]string{"cluster-autoscaler.kubernetes.io/safe-to-evict": "true"},
			expReplicas:            0,
		}},
	}, {
		name: "multiple override",
		override: []base.DeploymentOverride{
			{
				Name:         "controller",
				Labels:       map[string]string{"a": "b"},
				Annotations:  map[string]string{"c": "d"},
				Replicas:     &five,
				NodeSelector: map[string]string{"env": "dev"},
				Tolerations: []corev1.Toleration{{
					Key:      corev1.TaintNodeNotReady,
					Operator: corev1.TolerationOpExists,
					Effect:   corev1.TaintEffectNoSchedule,
				}},
				Affinity: &corev1.Affinity{
					NodeAffinity: &corev1.NodeAffinity{},
					PodAffinity:  &corev1.PodAffinity{},
					PodAntiAffinity: &corev1.PodAntiAffinity{
						PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
							{
								PodAffinityTerm: v1.PodAffinityTerm{
									Namespaces: []string{"test"},
								},
								Weight: 10,
							},
						},
					},
				},
			},
			{
				Name:         "webhook",
				Labels:       map[string]string{"e": "f"},
				Annotations:  map[string]string{"g": "h"},
				Replicas:     &four,
				NodeSelector: map[string]string{"env": "prod"},
				Tolerations: []corev1.Toleration{{
					Key:      corev1.TaintNodeUnschedulable,
					Operator: corev1.TolerationOpExists,
					Effect:   corev1.TaintEffectNoSchedule,
				}},
				Affinity: &corev1.Affinity{
					NodeAffinity: &corev1.NodeAffinity{},
					PodAffinity: &corev1.PodAffinity{
						PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
							{
								PodAffinityTerm: v1.PodAffinityTerm{
									Namespaces: []string{"test"},
								},
								Weight: 10,
							},
						},
					},
					PodAntiAffinity: &corev1.PodAntiAffinity{},
				},
			},
		},
		globalReplicas: 10,
		expDeployment: map[string]expDeployments{
			"controller": {
				expLabels:              map[string]string{"serving.knative.dev/release": "v0.13.0", "a": "b"},
				expTemplateLabels:      map[string]string{"serving.knative.dev/release": "v0.13.0", "app": "controller", "a": "b"},
				expAnnotations:         map[string]string{"c": "d"},
				expTemplateAnnotations: map[string]string{"cluster-autoscaler.kubernetes.io/safe-to-evict": "true", "c": "d"},
				expReplicas:            5,
				expNodeSelector:        map[string]string{"env": "dev"},
				expTolerations: []corev1.Toleration{{
					Key:      corev1.TaintNodeNotReady,
					Operator: corev1.TolerationOpExists,
					Effect:   corev1.TaintEffectNoSchedule,
				}},
				expAffinity: &corev1.Affinity{
					NodeAffinity: &corev1.NodeAffinity{},
					PodAffinity:  &corev1.PodAffinity{},
					PodAntiAffinity: &corev1.PodAntiAffinity{
						PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
							{
								PodAffinityTerm: v1.PodAffinityTerm{
									Namespaces: []string{"test"},
								},
								Weight: 10,
							},
						},
					},
				},
			},
			"webhook": {
				expLabels:              map[string]string{"serving.knative.dev/release": "v0.13.0", "e": "f"},
				expTemplateLabels:      map[string]string{"serving.knative.dev/release": "v0.13.0", "app": "webhook", "role": "webhook", "e": "f"},
				expAnnotations:         map[string]string{"g": "h"},
				expTemplateAnnotations: map[string]string{"cluster-autoscaler.kubernetes.io/safe-to-evict": "false", "g": "h"},
				expReplicas:            4,
				expNodeSelector:        map[string]string{"env": "prod"},
				expTolerations: []corev1.Toleration{{
					Key:      corev1.TaintNodeUnschedulable,
					Operator: corev1.TolerationOpExists,
					Effect:   corev1.TaintEffectNoSchedule,
				}},
				expAffinity: &corev1.Affinity{
					NodeAffinity: &corev1.NodeAffinity{},
					PodAffinity: &corev1.PodAffinity{
						PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
							{
								PodAffinityTerm: v1.PodAffinityTerm{
									Namespaces: []string{"test"},
								},
								Weight: 10,
							},
						},
					},
					PodAntiAffinity: &corev1.PodAntiAffinity{},
				},
			},
		},
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
						DeploymentOverride: test.override,
						HighAvailability: &base.HighAvailability{
							Replicas: &test.globalReplicas,
						},
					},
				},
			}

			//manifest, err = manifest.Transform(DeploymentsTransform(ks, log), HighAvailabilityTransform(ks, log))
			manifest, err = manifest.Transform(HighAvailabilityTransform(ks, log), DeploymentsTransform(ks, log))
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

						if diff := cmp.Diff(got.Spec.Template.Spec.NodeSelector, d.expNodeSelector); diff != "" {
							t.Fatalf("Unexpected nodeSelector: %v", diff)
						}

						if diff := cmp.Diff(got.Spec.Template.Spec.Tolerations, d.expTolerations); diff != "" {
							t.Fatalf("Unexpected tolerations: %v", diff)
						}

						if diff := cmp.Diff(got.Spec.Template.Spec.Affinity, d.expAffinity); diff != "" {
							t.Fatalf("Unexpected affinity: %v", diff)
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
					}
				}
			}
		})
	}
}

func TestDeploymentResourceRequirementsTransform(t *testing.T) {
	tests := []struct {
		DeployName string
		Input      servingv1beta1.KnativeServing
		Expected   map[string]v1.ResourceRequirements
	}{{
		DeployName: "net-istio-controller",
		Input: servingv1beta1.KnativeServing{
			ObjectMeta: metav1.ObjectMeta{
				Name: "specific-container-for-deployment",
			},
			Spec: servingv1beta1.KnativeServingSpec{
				CommonSpec: base.CommonSpec{
					Version: test.OperatorFlags.PreviousEventingVersion,
					DeploymentOverride: []base.DeploymentOverride{
						{
							Name: "net-istio-controller",
							Resources: []base.ResourceRequirementsOverride{{
								Container: "controller",
								ResourceRequirements: corev1.ResourceRequirements{
									Limits: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("999m"),
										corev1.ResourceMemory: resource.MustParse("999Mi")},
									Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("999m"),
										corev1.ResourceMemory: resource.MustParse("999Mi")},
								},
							}},
						},
					},
				},
			},
		},
		Expected: map[string]v1.ResourceRequirements{"controller": corev1.ResourceRequirements{
			Limits: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("999m"),
				corev1.ResourceMemory: resource.MustParse("999Mi")},
			Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("999m"),
				corev1.ResourceMemory: resource.MustParse("999Mi")},
		}},
	}}

	for _, test := range tests {
		t.Run(test.Input.Name, func(t *testing.T) {
			manifest, err := mf.NewManifest("testdata/manifest.yaml")
			if err != nil {
				t.Fatalf("Failed to create manifest: %v", err)
			}
			actual, err := manifest.Transform(DeploymentsTransform(&test.Input, log))
			if err != nil {
				t.Fatalf("Failed to transform manifest: %v", err)
			}
			resources := actual.Filter(mf.ByKind("Deployment")).Filter(mf.ByName(test.DeployName)).Resources()
			util.AssertEqual(t, len(resources), 1)
			deployment := &appsv1.Deployment{}
			if err = scheme.Scheme.Convert(&resources[0], deployment, nil); err != nil {
				t.Fatalf("Failed to convert unstructured to deployment: %v", err)
			}
			containers := deployment.Spec.Template.Spec.Containers
			for i := range containers {
				expected := test.Expected[containers[i].Name]
				if !reflect.DeepEqual(containers[i].Resources, expected) {
					t.Errorf("\n    Name: %s\n  Expect: %v\n  Actual: %v", containers[i].Name, expected, containers[i].Resources)
				}
			}
		})
	}
}
