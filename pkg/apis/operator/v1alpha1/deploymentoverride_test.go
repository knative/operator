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

package v1alpha1

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"knative.dev/operator/pkg/apis/operator/base"
	util "knative.dev/operator/pkg/reconciler/common/testing"
)

func makeExpectedDeploymentOverrideNonEmpty() []base.DeploymentOverride {
	return []base.DeploymentOverride{
		{
			Name:         "webhook",
			Labels:       map[string]string{"e": "f"},
			Annotations:  map[string]string{"g": "h"},
			Replicas:     4,
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
		}, {
			Name: "webhook-not-exist",
			Resources: []base.ResourceRequirementsOverride{{
				Container: "webhook-not-exist",
				ResourceRequirements: corev1.ResourceRequirements{
					Limits: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("999m"),
						corev1.ResourceMemory: resource.MustParse("999Mi")},
					Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("999m"),
						corev1.ResourceMemory: resource.MustParse("999Mi")},
				},
			}},
		},
	}
}

func makeExpectedExistentTestDeploymentOverride() []base.DeploymentOverride {
	return []base.DeploymentOverride{
		{
			Name:         "webhook",
			Labels:       map[string]string{"e": "f"},
			Annotations:  map[string]string{"g": "h"},
			Replicas:     4,
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
			Resources: []base.ResourceRequirementsOverride{{
				Container: "webhook",
				ResourceRequirements: corev1.ResourceRequirements{
					Limits: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("999m"),
						corev1.ResourceMemory: resource.MustParse("999Mi")},
					Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("999m"),
						corev1.ResourceMemory: resource.MustParse("999Mi")},
				},
			}},
		},
	}
}

func makeExpectedDeploymentOverrideArrayOrigin() []base.DeploymentOverride {
	return []base.DeploymentOverride{
		{
			Name:         "webhook",
			Labels:       map[string]string{"e": "f"},
			Annotations:  map[string]string{"g": "h"},
			Replicas:     4,
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
			Resources: []base.ResourceRequirementsOverride{{
				Container: "webhook",
				ResourceRequirements: corev1.ResourceRequirements{
					Limits: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("99m"),
						corev1.ResourceMemory: resource.MustParse("99Mi")},
					Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("99m"),
						corev1.ResourceMemory: resource.MustParse("99Mi")},
				},
			}},
		},
	}
}

func makeExpectedDeploymentOverrideEmpty() []base.DeploymentOverride {
	return []base.DeploymentOverride{
		{
			Name: "webhook-not-exist",
			Resources: []base.ResourceRequirementsOverride{{
				Container: "webhook-not-exist",
				ResourceRequirements: corev1.ResourceRequirements{
					Limits: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("999m"),
						corev1.ResourceMemory: resource.MustParse("999Mi")},
					Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("999m"),
						corev1.ResourceMemory: resource.MustParse("999Mi")},
				},
			}},
		},
	}
}

func TestConvertToDeploymentOverride(t *testing.T) {
	expectedDeploymentOverrideEmpty := makeExpectedDeploymentOverrideEmpty()
	expectedDeploymentOverrideNonEmpty := makeExpectedDeploymentOverrideNonEmpty()
	expectedExistentDeploymentOverrideArray := makeExpectedExistentTestDeploymentOverride()
	expectedDeploymentOverrideArrayOrigin := makeExpectedDeploymentOverrideArrayOrigin()
	for _, tt := range []struct {
		name     string
		source   base.KComponent
		expected []base.DeploymentOverride
	}{{
		name: "Knative Serving: merge the non-existent ResourceRequirementsOverride into the non-empty DeploymentOverrides",
		source: &KnativeServing{
			Spec: KnativeServingSpec{
				CommonSpec: base.CommonSpec{
					DeprecatedResources: []base.ResourceRequirementsOverride{{
						Container: "webhook-not-exist",
						ResourceRequirements: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("999m"),
								corev1.ResourceMemory: resource.MustParse("999Mi")},
							Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("999m"),
								corev1.ResourceMemory: resource.MustParse("999Mi")},
						},
					}},
					DeploymentOverride: []base.DeploymentOverride{
						{
							Name:         "webhook",
							Labels:       map[string]string{"e": "f"},
							Annotations:  map[string]string{"g": "h"},
							Replicas:     4,
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
				},
			},
		},
		expected: expectedDeploymentOverrideNonEmpty,
	}, {
		name: "Knative Serving: merge the existent ResourceRequirementsOverride into the non-empty DeploymentOverrides",
		source: &KnativeServing{
			Spec: KnativeServingSpec{
				CommonSpec: base.CommonSpec{
					DeprecatedResources: []base.ResourceRequirementsOverride{{
						Container: "webhook",
						ResourceRequirements: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("999m"),
								corev1.ResourceMemory: resource.MustParse("999Mi")},
							Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("999m"),
								corev1.ResourceMemory: resource.MustParse("999Mi")},
						},
					}},
					DeploymentOverride: []base.DeploymentOverride{
						{
							Name:         "webhook",
							Labels:       map[string]string{"e": "f"},
							Annotations:  map[string]string{"g": "h"},
							Replicas:     4,
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
				},
			},
		},
		expected: expectedExistentDeploymentOverrideArray,
	}, {
		name: "Knative Serving: not merge the ResourceRequirementsOverride into the DeploymentOverrides due to existing DeploymentOverrides",
		source: &KnativeServing{
			Spec: KnativeServingSpec{
				CommonSpec: base.CommonSpec{
					DeprecatedResources: []base.ResourceRequirementsOverride{{
						Container: "webhook",
						ResourceRequirements: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("999m"),
								corev1.ResourceMemory: resource.MustParse("999Mi")},
							Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("999m"),
								corev1.ResourceMemory: resource.MustParse("999Mi")},
						},
					}},
					DeploymentOverride: []base.DeploymentOverride{
						{
							Name:         "webhook",
							Labels:       map[string]string{"e": "f"},
							Annotations:  map[string]string{"g": "h"},
							Replicas:     4,
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
							Resources: []base.ResourceRequirementsOverride{{
								Container: "webhook",
								ResourceRequirements: corev1.ResourceRequirements{
									Limits: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("99m"),
										corev1.ResourceMemory: resource.MustParse("99Mi")},
									Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("99m"),
										corev1.ResourceMemory: resource.MustParse("99Mi")},
								},
							}},
						},
					},
				},
			},
		},
		expected: expectedDeploymentOverrideArrayOrigin,
	}, {
		name: "Knative Serving: not merge the ResourceRequirementsOverride into the DeploymentOverrides due to empty DeploymentOverrides",
		source: &KnativeServing{
			Spec: KnativeServingSpec{
				CommonSpec: base.CommonSpec{
					DeprecatedResources: []base.ResourceRequirementsOverride{{
						Container: "webhook",
						ResourceRequirements: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("999m"),
								corev1.ResourceMemory: resource.MustParse("999Mi")},
							Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("999m"),
								corev1.ResourceMemory: resource.MustParse("999Mi")},
						},
					}},
					DeploymentOverride: []base.DeploymentOverride{
						{
							Name:         "webhook",
							Labels:       map[string]string{"e": "f"},
							Annotations:  map[string]string{"g": "h"},
							Replicas:     4,
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
							Resources: []base.ResourceRequirementsOverride{{
								Container: "webhook",
							}},
						},
					},
				},
			},
		},
		expected: expectedExistentDeploymentOverrideArray,
	}, {
		name: "Knative Serving: merge the non-existent ResourceRequirementsOverride into the DeploymentOverrides",
		source: &KnativeServing{
			Spec: KnativeServingSpec{
				CommonSpec: base.CommonSpec{
					DeprecatedResources: []base.ResourceRequirementsOverride{{
						Container: "webhook-not-exist",
						ResourceRequirements: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("999m"),
								corev1.ResourceMemory: resource.MustParse("999Mi")},
							Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("999m"),
								corev1.ResourceMemory: resource.MustParse("999Mi")},
						},
					}},
					DeploymentOverride: []base.DeploymentOverride{
						{
							Name:         "webhook",
							Labels:       map[string]string{"e": "f"},
							Annotations:  map[string]string{"g": "h"},
							Replicas:     4,
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
				},
			},
		},
		expected: expectedDeploymentOverrideNonEmpty,
	}, {
		name: "Knative Serving: merge the non-existent ResourceRequirementsOverride into the DeploymentOverrides",
		source: &KnativeServing{
			Spec: KnativeServingSpec{
				CommonSpec: base.CommonSpec{
					DeprecatedResources: []base.ResourceRequirementsOverride{{
						Container: "webhook-not-exist",
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
		expected: expectedDeploymentOverrideEmpty,
	}, {
		name: "Knative Serving: merge the existent ResourceRequirementsOverride into the DeploymentOverrides",
		source: &KnativeServing{
			Spec: KnativeServingSpec{
				CommonSpec: base.CommonSpec{
					DeprecatedResources: []base.ResourceRequirementsOverride{{
						Container: "webhook",
						ResourceRequirements: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("999m"),
								corev1.ResourceMemory: resource.MustParse("999Mi")},
							Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("999m"),
								corev1.ResourceMemory: resource.MustParse("999Mi")},
						},
					}},
					DeploymentOverride: []base.DeploymentOverride{
						{
							Name:         "webhook",
							Labels:       map[string]string{"e": "f"},
							Annotations:  map[string]string{"g": "h"},
							Replicas:     4,
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
				},
			},
		},
		expected: expectedExistentDeploymentOverrideArray,
	}} {

		t.Run(tt.name, func(t *testing.T) {
			deploymentOverrides := ConvertToDeploymentOverride(tt.source)
			util.AssertDeepEqual(t, deploymentOverrides, tt.expected)
		})
	}
}
