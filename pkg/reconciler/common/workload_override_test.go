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
	"google.golang.org/api/googleapi"
	appsv1 "k8s.io/api/apps/v1"
	v2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"

	"knative.dev/operator/pkg/apis/operator/base"
	servingv1beta1 "knative.dev/operator/pkg/apis/operator/v1beta1"
	util "knative.dev/operator/pkg/reconciler/common/testing"
	"knative.dev/operator/test"
	"knative.dev/pkg/ptr"
)

type expDeployments struct {
	expLabels                    map[string]string
	expTemplateLabels            map[string]string
	expAnnotations               map[string]string
	expTemplateAnnotations       map[string]string
	expReplicas                  int32
	expNodeSelector              map[string]string
	expTopologySpreadConstraints []corev1.TopologySpreadConstraint
	expTolerations               []corev1.Toleration
	expAffinity                  *corev1.Affinity
	expEnv                       map[string][]corev1.EnvVar
	expReadinessProbe            *corev1.Probe
	expLivenessProbe             *corev1.Probe
	expHostNetwork               *bool
	expDNSPolicy                 *corev1.DNSPolicy
}

type expHorizontalPodAutoscalers struct {
	expMinReplicas int32
	expMaxReplicas int32
}

type expJobs struct {
	expNodeSelector map[string]string
	expTolerations  []corev1.Toleration
}

func TestComponentsTransform(t *testing.T) {
	var four int32 = 4
	var five int32 = 5
	var defaultDnsPolicy = corev1.DNSPolicy("")
	var dnsClusterFirstWithHostNet = corev1.DNSClusterFirstWithHostNet
	tests := []struct {
		name                       string
		override                   []base.WorkloadOverride
		globalReplicas             int32
		expDeployment              map[string]expDeployments
		expHorizontalPodAutoscaler map[string]expHorizontalPodAutoscalers
	}{{
		name:     "no override",
		override: nil,
		expDeployment: map[string]expDeployments{"controller": {
			expLabels:                    map[string]string{"serving.knative.dev/release": "v0.13.0"},
			expTemplateLabels:            map[string]string{"serving.knative.dev/release": "v0.13.0", "app": "controller"},
			expAnnotations:               nil,
			expTemplateAnnotations:       map[string]string{"cluster-autoscaler.kubernetes.io/safe-to-evict": "true"},
			expReplicas:                  0,
			expNodeSelector:              nil,
			expTopologySpreadConstraints: nil,
			expTolerations:               nil,
			expAffinity:                  nil,
			expHostNetwork:               nil,
			expDNSPolicy:                 nil,
		}},
	}, {
		name: "simple override",
		override: []base.WorkloadOverride{
			{
				Name:         "controller",
				Labels:       map[string]string{"a": "b"},
				Annotations:  map[string]string{"c": "d"},
				Replicas:     &five,
				NodeSelector: map[string]string{"env": "prod"},
				TopologySpreadConstraints: []corev1.TopologySpreadConstraint{{
					MaxSkew:           1,
					TopologyKey:       corev1.LabelTopologyZone,
					WhenUnsatisfiable: corev1.DoNotSchedule,
					LabelSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "controller"},
					},
				}},
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
				HostNetwork: googleapi.Bool(false),
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
			expTopologySpreadConstraints: []corev1.TopologySpreadConstraint{{
				MaxSkew:           1,
				TopologyKey:       corev1.LabelTopologyZone,
				WhenUnsatisfiable: corev1.DoNotSchedule,
				LabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "controller"},
				},
			}},
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
			expHostNetwork: googleapi.Bool(false),
			expDNSPolicy:   nil,
		}},
	}, {
		name: "no replicas in workload override, use global replicas",
		override: []base.WorkloadOverride{
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
		name: "unset host network",
		override: []base.WorkloadOverride{
			{
				Name:        "controller",
				HostNetwork: nil,
			},
		},
		globalReplicas: 10,
		expDeployment: map[string]expDeployments{"controller": {
			expLabels:              map[string]string{"serving.knative.dev/release": "v0.13.0"},
			expTemplateLabels:      map[string]string{"serving.knative.dev/release": "v0.13.0", "app": "controller"},
			expTemplateAnnotations: map[string]string{"cluster-autoscaler.kubernetes.io/safe-to-evict": "true"},
			expReplicas:            10,
			expHostNetwork:         nil,
		}},
	}, {
		name: "host network is true",
		override: []base.WorkloadOverride{
			{
				Name:        "controller",
				HostNetwork: googleapi.Bool(true),
			},
		},
		globalReplicas: 10,
		expDeployment: map[string]expDeployments{"controller": {
			expLabels:              map[string]string{"serving.knative.dev/release": "v0.13.0"},
			expTemplateLabels:      map[string]string{"serving.knative.dev/release": "v0.13.0", "app": "controller"},
			expTemplateAnnotations: map[string]string{"cluster-autoscaler.kubernetes.io/safe-to-evict": "true"},
			expReplicas:            10,
			expHostNetwork:         googleapi.Bool(true),
			expDNSPolicy:           &dnsClusterFirstWithHostNet,
		}},
	}, {
		name: "host network is false",
		override: []base.WorkloadOverride{
			{
				Name:        "controller",
				HostNetwork: googleapi.Bool(false),
			},
		},
		globalReplicas: 10,
		expDeployment: map[string]expDeployments{"controller": {
			expLabels:              map[string]string{"serving.knative.dev/release": "v0.13.0"},
			expTemplateLabels:      map[string]string{"serving.knative.dev/release": "v0.13.0", "app": "controller"},
			expTemplateAnnotations: map[string]string{"cluster-autoscaler.kubernetes.io/safe-to-evict": "true"},
			expReplicas:            10,
			expHostNetwork:         googleapi.Bool(false),
			expDNSPolicy:           nil,
		}},
	}, {
		name: "override env vars",
		override: []base.WorkloadOverride{
			{
				Name: "controller",
				Env: []base.EnvRequirementsOverride{{
					Container: "controller",
					EnvVars: []corev1.EnvVar{{
						Name:  "METRICS_DOMAIN",
						Value: "test",
					}},
				}},
			},
		},
		globalReplicas: 10,
		expDeployment: map[string]expDeployments{"controller": {
			expLabels:              map[string]string{"serving.knative.dev/release": "v0.13.0"},
			expTemplateLabels:      map[string]string{"serving.knative.dev/release": "v0.13.0", "app": "controller"},
			expTemplateAnnotations: map[string]string{"cluster-autoscaler.kubernetes.io/safe-to-evict": "true"},
			expReplicas:            10,
			expEnv: map[string][]corev1.EnvVar{"controller": {
				{
					Name: "SYSTEM_NAMESPACE",
					ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{
							FieldPath: "metadata.namespace",
						}},
				}, {
					Name:  "CONFIG_LOGGING_NAME",
					Value: "config-logging",
				}, {
					Name:  "CONFIG_OBSERVABILITY_NAME",
					Value: "config-observability",
				}, {
					Name:  "METRICS_DOMAIN",
					Value: "test",
				}}},
		}},
	}, {
		name: "duplicate env vars overrides are applied multiple times on existing env var",
		override: []base.WorkloadOverride{
			{
				Name: "controller",
				Env: []base.EnvRequirementsOverride{{
					Container: "controller",
					EnvVars: []corev1.EnvVar{{
						Name:  "METRICS_DOMAIN",
						Value: "test1",
					}, {
						Name:  "METRICS_DOMAIN",
						Value: "test2",
					}},
				}},
			},
		},
		globalReplicas: 10,
		expDeployment: map[string]expDeployments{"controller": {
			expLabels:              map[string]string{"serving.knative.dev/release": "v0.13.0"},
			expTemplateLabels:      map[string]string{"serving.knative.dev/release": "v0.13.0", "app": "controller"},
			expTemplateAnnotations: map[string]string{"cluster-autoscaler.kubernetes.io/safe-to-evict": "true"},
			expReplicas:            10,
			expEnv: map[string][]corev1.EnvVar{"controller": {
				{
					Name: "SYSTEM_NAMESPACE",
					ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{
							FieldPath: "metadata.namespace",
						}},
				}, {
					Name:  "CONFIG_LOGGING_NAME",
					Value: "config-logging",
				}, {
					Name:  "CONFIG_OBSERVABILITY_NAME",
					Value: "config-observability",
				}, {
					Name:  "METRICS_DOMAIN",
					Value: "test2",
				}}},
		}},
	}, {
		name: "env var overriding has no effect if container name does not exist",
		override: []base.WorkloadOverride{
			{
				Name: "controller",
				Env: []base.EnvRequirementsOverride{{
					Container: "wrong_name",
					EnvVars: []corev1.EnvVar{{
						Name:  "METRICS_DOMAIN",
						Value: "test1",
					}, {
						Name:  "METRICS_DOMAIN",
						Value: "test2",
					}},
				}},
			},
		},
		globalReplicas: 10,
		expDeployment: map[string]expDeployments{"controller": {
			expLabels:              map[string]string{"serving.knative.dev/release": "v0.13.0"},
			expTemplateLabels:      map[string]string{"serving.knative.dev/release": "v0.13.0", "app": "controller"},
			expTemplateAnnotations: map[string]string{"cluster-autoscaler.kubernetes.io/safe-to-evict": "true"},
			expReplicas:            10,
			expEnv: map[string][]corev1.EnvVar{"controller": {
				{
					Name: "SYSTEM_NAMESPACE",
					ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{
							FieldPath: "metadata.namespace",
						}},
				}, {
					Name:  "CONFIG_LOGGING_NAME",
					Value: "config-logging",
				}, {
					Name:  "CONFIG_OBSERVABILITY_NAME",
					Value: "config-observability",
				}, {
					Name:  "METRICS_DOMAIN",
					Value: "knative.dev/internal/serving",
				}}},
		}},
	}, {
		name: "add env var via overriding",
		override: []base.WorkloadOverride{
			{
				Name: "controller",
				Env: []base.EnvRequirementsOverride{{
					Container: "controller",
					EnvVars: []corev1.EnvVar{{
						Name:  "TEST_ENV",
						Value: "test1",
					}},
				}},
			},
		},
		globalReplicas: 10,
		expDeployment: map[string]expDeployments{"controller": {
			expLabels:              map[string]string{"serving.knative.dev/release": "v0.13.0"},
			expTemplateLabels:      map[string]string{"serving.knative.dev/release": "v0.13.0", "app": "controller"},
			expTemplateAnnotations: map[string]string{"cluster-autoscaler.kubernetes.io/safe-to-evict": "true"},
			expReplicas:            10,
			expEnv: map[string][]corev1.EnvVar{"controller": {
				{
					Name: "SYSTEM_NAMESPACE",
					ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{
							FieldPath: "metadata.namespace",
						}},
				}, {
					Name:  "CONFIG_LOGGING_NAME",
					Value: "config-logging",
				}, {
					Name:  "CONFIG_OBSERVABILITY_NAME",
					Value: "config-observability",
				}, {
					Name:  "METRICS_DOMAIN",
					Value: "knative.dev/internal/serving",
				}, {
					Name:  "TEST_ENV",
					Value: "test1",
				}}},
		}},
	}, {
		name: "add env var via overriding and modify an existing one",
		override: []base.WorkloadOverride{
			{
				Name: "controller",
				Env: []base.EnvRequirementsOverride{{
					Container: "controller",
					EnvVars: []corev1.EnvVar{{
						Name:  "TEST_ENV",
						Value: "test1",
					}, {
						Name:  "METRICS_DOMAIN",
						Value: "test1",
					}},
				}},
			},
		},
		globalReplicas: 10,
		expDeployment: map[string]expDeployments{"controller": {
			expLabels:              map[string]string{"serving.knative.dev/release": "v0.13.0"},
			expTemplateLabels:      map[string]string{"serving.knative.dev/release": "v0.13.0", "app": "controller"},
			expTemplateAnnotations: map[string]string{"cluster-autoscaler.kubernetes.io/safe-to-evict": "true"},
			expReplicas:            10,
			expEnv: map[string][]corev1.EnvVar{"controller": {
				{
					Name: "SYSTEM_NAMESPACE",
					ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{
							FieldPath: "metadata.namespace",
						}},
				}, {
					Name:  "CONFIG_LOGGING_NAME",
					Value: "config-logging",
				}, {
					Name:  "CONFIG_OBSERVABILITY_NAME",
					Value: "config-observability",
				}, {
					Name:  "METRICS_DOMAIN",
					Value: "test1",
				}, {
					Name:  "TEST_ENV",
					Value: "test1",
				}}},
		}},
	}, {
		name: "duplicate added env vars are overwritten",
		override: []base.WorkloadOverride{
			{
				Name: "controller",
				Env: []base.EnvRequirementsOverride{{
					Container: "controller",
					EnvVars: []corev1.EnvVar{{
						Name:  "TEST_ENV",
						Value: "test1",
					}, {
						Name:  "TEST_ENV",
						Value: "test2",
					}},
				}},
			},
		},
		globalReplicas: 10,
		expDeployment: map[string]expDeployments{"controller": {
			expLabels:              map[string]string{"serving.knative.dev/release": "v0.13.0"},
			expTemplateLabels:      map[string]string{"serving.knative.dev/release": "v0.13.0", "app": "controller"},
			expTemplateAnnotations: map[string]string{"cluster-autoscaler.kubernetes.io/safe-to-evict": "true"},
			expReplicas:            10,
			expEnv: map[string][]corev1.EnvVar{"controller": {
				{
					Name: "SYSTEM_NAMESPACE",
					ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{
							FieldPath: "metadata.namespace",
						}},
				}, {
					Name:  "CONFIG_LOGGING_NAME",
					Value: "config-logging",
				}, {
					Name:  "CONFIG_OBSERVABILITY_NAME",
					Value: "config-observability",
				}, {
					Name:  "METRICS_DOMAIN",
					Value: "knative.dev/internal/serving",
				}, {
					Name:  "TEST_ENV",
					Value: "test2",
				}}},
		}},
	}, {
		name: "simple probe overrides",
		override: []base.WorkloadOverride{
			{
				Name: "activator",
				ReadinessProbes: []base.ProbesRequirementsOverride{{
					Container:           "activator",
					TimeoutSeconds:      15,
					InitialDelaySeconds: 12,
					SuccessThreshold:    3,
				}},
				LivenessProbes: []base.ProbesRequirementsOverride{{
					Container:           "activator",
					TimeoutSeconds:      4,
					InitialDelaySeconds: 2,
				}},
			},
		},
		expDeployment: map[string]expDeployments{"activator": {
			expLabels:              map[string]string{"serving.knative.dev/release": "v0.13.0"},
			expTemplateLabels:      map[string]string{"serving.knative.dev/release": "v0.13.0", "app": "activator", "role": "activator"},
			expTemplateAnnotations: map[string]string{"cluster-autoscaler.kubernetes.io/safe-to-evict": "false"},
			expReplicas:            0,
			expReadinessProbe: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Port:        intstr.IntOrString{IntVal: 8012},
						HTTPHeaders: []corev1.HTTPHeader{{Name: "k-kubelet-probe", Value: "activator"}},
					}},
				TimeoutSeconds:      15,
				InitialDelaySeconds: 12,
				SuccessThreshold:    3,
			},
			expLivenessProbe: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Port:        intstr.IntOrString{IntVal: 8012},
						HTTPHeaders: []corev1.HTTPHeader{{Name: "k-kubelet-probe", Value: "activator"}},
					}},
				TimeoutSeconds:      4,
				InitialDelaySeconds: 2,
			},
		}},
	}, {
		name: "override nil probe",
		override: []base.WorkloadOverride{
			{
				Name: "controller",
				ReadinessProbes: []base.ProbesRequirementsOverride{{
					Container:           "controller",
					TimeoutSeconds:      15,
					InitialDelaySeconds: 12,
				}},
			},
		},
		expDeployment: map[string]expDeployments{"controller": {
			expLabels:              map[string]string{"serving.knative.dev/release": "v0.13.0"},
			expTemplateLabels:      map[string]string{"serving.knative.dev/release": "v0.13.0", "app": "controller"},
			expTemplateAnnotations: map[string]string{"cluster-autoscaler.kubernetes.io/safe-to-evict": "true"},
			expReplicas:            0,
			expReadinessProbe: &corev1.Probe{
				TimeoutSeconds:      15,
				InitialDelaySeconds: 12,
			}}},
	}, {
		name: "empty readiness probe drops probe",
		override: []base.WorkloadOverride{
			{
				Name: "activator",
				ReadinessProbes: []base.ProbesRequirementsOverride{{
					Container: "activator",
				}}},
		},
		expDeployment: map[string]expDeployments{"activator": {
			expLabels:              map[string]string{"serving.knative.dev/release": "v0.13.0"},
			expTemplateLabels:      map[string]string{"serving.knative.dev/release": "v0.13.0", "app": "activator", "role": "activator"},
			expTemplateAnnotations: map[string]string{"cluster-autoscaler.kubernetes.io/safe-to-evict": "false"},
			expReplicas:            0,
			expLivenessProbe: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Port:        intstr.IntOrString{IntVal: 8012},
						HTTPHeaders: []corev1.HTTPHeader{{Name: "k-kubelet-probe", Value: "activator"}},
					}},
			},
		}},
	}, {
		name: "empty liveness probe drops probe",
		override: []base.WorkloadOverride{
			{
				Name: "activator",
				LivenessProbes: []base.ProbesRequirementsOverride{{
					Container: "activator",
				}}},
		},
		expDeployment: map[string]expDeployments{"activator": {
			expLabels:              map[string]string{"serving.knative.dev/release": "v0.13.0"},
			expTemplateLabels:      map[string]string{"serving.knative.dev/release": "v0.13.0", "app": "activator", "role": "activator"},
			expTemplateAnnotations: map[string]string{"cluster-autoscaler.kubernetes.io/safe-to-evict": "false"},
			expReplicas:            0,
			expReadinessProbe: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Port:        intstr.IntOrString{IntVal: 8012},
						HTTPHeaders: []corev1.HTTPHeader{{Name: "k-kubelet-probe", Value: "activator"}},
					}},
			},
		}},
	}, {
		name: "neither replicas in workload override nor global replicas",
		override: []base.WorkloadOverride{
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
		override: []base.WorkloadOverride{
			{
				Name:         "controller",
				Labels:       map[string]string{"a": "b"},
				Annotations:  map[string]string{"c": "d"},
				Replicas:     &five,
				NodeSelector: map[string]string{"env": "dev"},
				TopologySpreadConstraints: []corev1.TopologySpreadConstraint{{
					MaxSkew:           1,
					TopologyKey:       corev1.LabelTopologyZone,
					WhenUnsatisfiable: corev1.DoNotSchedule,
					LabelSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "controller"},
					},
				}},
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
								PodAffinityTerm: corev1.PodAffinityTerm{
									Namespaces: []string{"test"},
								},
								Weight: 10,
							},
						},
					},
				},
				HostNetwork: googleapi.Bool(false),
			},
			{
				Name:         "webhook",
				Labels:       map[string]string{"e": "f"},
				Annotations:  map[string]string{"g": "h"},
				Replicas:     &four,
				NodeSelector: map[string]string{"env": "prod"},
				TopologySpreadConstraints: []corev1.TopologySpreadConstraint{{
					MaxSkew:           1,
					TopologyKey:       corev1.LabelTopologyZone,
					WhenUnsatisfiable: corev1.DoNotSchedule,
					LabelSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "controller"},
					},
				}},
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
								PodAffinityTerm: corev1.PodAffinityTerm{
									Namespaces: []string{"test"},
								},
								Weight: 10,
							},
						},
					},
					PodAntiAffinity: &corev1.PodAntiAffinity{},
				},
				HostNetwork: googleapi.Bool(true),
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
				expTopologySpreadConstraints: []corev1.TopologySpreadConstraint{{
					MaxSkew:           1,
					TopologyKey:       corev1.LabelTopologyZone,
					WhenUnsatisfiable: corev1.DoNotSchedule,
					LabelSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "controller"},
					},
				}},
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
								PodAffinityTerm: corev1.PodAffinityTerm{
									Namespaces: []string{"test"},
								},
								Weight: 10,
							},
						},
					},
				},
				expHostNetwork: googleapi.Bool(false),
				expDNSPolicy:   nil,
			},
			"webhook": {
				expLabels:              map[string]string{"serving.knative.dev/release": "v0.13.0", "e": "f"},
				expTemplateLabels:      map[string]string{"serving.knative.dev/release": "v0.13.0", "app": "webhook", "role": "webhook", "e": "f"},
				expAnnotations:         map[string]string{"g": "h"},
				expTemplateAnnotations: map[string]string{"cluster-autoscaler.kubernetes.io/safe-to-evict": "false", "g": "h"},
				expReplicas:            0,
				expNodeSelector:        map[string]string{"env": "prod"},
				expTopologySpreadConstraints: []corev1.TopologySpreadConstraint{{
					MaxSkew:           1,
					TopologyKey:       corev1.LabelTopologyZone,
					WhenUnsatisfiable: corev1.DoNotSchedule,
					LabelSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "controller"},
					},
				}},
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
								PodAffinityTerm: corev1.PodAffinityTerm{
									Namespaces: []string{"test"},
								},
								Weight: 10,
							},
						},
					},
					PodAntiAffinity: &corev1.PodAntiAffinity{},
				},
				expHostNetwork: googleapi.Bool(true),
				expDNSPolicy:   &dnsClusterFirstWithHostNet,
			},
		},
	}, {
		name: "activator HPA no override",
		expDeployment: map[string]expDeployments{
			"activator": {
				expLabels:              map[string]string{"serving.knative.dev/release": "v0.13.0"},
				expTemplateLabels:      map[string]string{"serving.knative.dev/release": "v0.13.0", "app": "activator", "role": "activator"},
				expAnnotations:         nil,
				expTemplateAnnotations: map[string]string{"cluster-autoscaler.kubernetes.io/safe-to-evict": "false"},
				expReplicas:            0, // if hpa is used, this should never be set
			},
		},
		expHorizontalPodAutoscaler: map[string]expHorizontalPodAutoscalers{
			"activator": {
				expMinReplicas: 1,  // defined in manifest.yaml
				expMaxReplicas: 20, // defined in manifest.yaml
			},
		},
	}, {
		name:           "activator HPA global replicas override",
		globalReplicas: 10,
		expDeployment: map[string]expDeployments{
			"activator": {
				expLabels:              map[string]string{"serving.knative.dev/release": "v0.13.0"},
				expTemplateLabels:      map[string]string{"serving.knative.dev/release": "v0.13.0", "app": "activator", "role": "activator"},
				expAnnotations:         nil,
				expTemplateAnnotations: map[string]string{"cluster-autoscaler.kubernetes.io/safe-to-evict": "false"},
				expReplicas:            0, // if hpa is used, this should never be set
			},
		},
		expHorizontalPodAutoscaler: map[string]expHorizontalPodAutoscalers{
			"activator": {
				expMinReplicas: 10,
				expMaxReplicas: 29, // in manifest.yaml maxReplicas=20 +9 (difference between existing min and overwritten min)
			},
		},
	}, {
		name: "activator HPA workload override",
		override: []base.WorkloadOverride{
			{
				Name:     "activator",
				Replicas: &four,
			},
		},
		expDeployment: map[string]expDeployments{
			"activator": {
				expLabels:              map[string]string{"serving.knative.dev/release": "v0.13.0"},
				expTemplateLabels:      map[string]string{"serving.knative.dev/release": "v0.13.0", "app": "activator", "role": "activator"},
				expAnnotations:         nil,
				expTemplateAnnotations: map[string]string{"cluster-autoscaler.kubernetes.io/safe-to-evict": "false"},
				expReplicas:            0, // if hpa is used, this should never be set
			},
		},
		expHorizontalPodAutoscaler: map[string]expHorizontalPodAutoscalers{
			"activator": {
				expMinReplicas: four,
				expMaxReplicas: 23, // in manifest.yaml maxReplicas=20 +3 (difference between existing min and overwritten min)
			},
		},
	}, {
		name:           "activator HPA global and workload override",
		globalReplicas: 10,
		override: []base.WorkloadOverride{
			{
				Name:     "activator",
				Replicas: &four,
			},
		},
		expDeployment: map[string]expDeployments{
			"activator": {
				expLabels:              map[string]string{"serving.knative.dev/release": "v0.13.0"},
				expTemplateLabels:      map[string]string{"serving.knative.dev/release": "v0.13.0", "app": "activator", "role": "activator"},
				expAnnotations:         nil,
				expTemplateAnnotations: map[string]string{"cluster-autoscaler.kubernetes.io/safe-to-evict": "false"},
				expReplicas:            0, // if hpa is used, this should never be set
			},
		},
		expHorizontalPodAutoscaler: map[string]expHorizontalPodAutoscalers{
			"activator": {
				expMinReplicas: four,
				expMaxReplicas: 23, // in manifest.yaml maxReplicas=20 +3 (difference between existing min and overwritten min)
			},
		},
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manifest, err := mf.NewManifest("testdata/manifest.yaml")
			if err != nil {
				t.Fatalf("Failed to create manifest: %v", err)
			}

			kss := map[string]*servingv1beta1.KnativeServing{
				"deprecated deployments": {
					Spec: servingv1beta1.KnativeServingSpec{
						CommonSpec: base.CommonSpec{
							DeploymentOverride: test.override,
							HighAvailability: &base.HighAvailability{
								Replicas: &test.globalReplicas,
							},
						},
					},
				},
				"components": {
					Spec: servingv1beta1.KnativeServingSpec{
						CommonSpec: base.CommonSpec{
							Workloads: test.override,
							HighAvailability: &base.HighAvailability{
								Replicas: &test.globalReplicas,
							},
						},
					},
				},
			}
			for key, ks := range kss {
				t.Run(key, func(t *testing.T) {

					manifest, err = manifest.Transform(HighAvailabilityTransform(ks), OverridesTransform(ks.GetSpec().GetWorkloadOverrides(), log))
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

								if diff := cmp.Diff(got.Spec.Template.Spec.TopologySpreadConstraints, d.expTopologySpreadConstraints); diff != "" {
									t.Fatalf("Unexpected topologySpreadConstraints: %v", diff)
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
								for c, envPerContainer := range d.expEnv {
									if diff := cmp.Diff(getEnv(got.Spec.Template.Spec.Containers, c), envPerContainer); diff != "" {
										t.Fatalf("Unexpected env in pod template: %v", diff)
									}
								}
								r, l := getProbes(got.Spec.Template.Spec.Containers, expName)
								if d.expReadinessProbe != nil {
									if diff := cmp.Diff(*r, *d.expReadinessProbe); diff != "" {
										t.Fatalf("Unexpected readiness probe in pod template: %v", diff)
									}
								}
								if d.expLivenessProbe != nil {
									if diff := cmp.Diff(*l, *d.expLivenessProbe); diff != "" {
										t.Fatalf("Unexpected liveness probe in pod template: %v", diff)
									}
								}
								hostNetwork := googleapi.Bool(false)
								if d.expHostNetwork != nil {
									hostNetwork = d.expHostNetwork
								}
								if diff := cmp.Diff(&got.Spec.Template.Spec.HostNetwork, hostNetwork); diff != "" {
									t.Fatalf("Unexpected hostNetwork: %v", diff)
								}
								dnsPolicy := &defaultDnsPolicy
								if d.expDNSPolicy != nil {
									dnsPolicy = d.expDNSPolicy
								}
								if diff := cmp.Diff(&got.Spec.Template.Spec.DNSPolicy, dnsPolicy); diff != "" {
									t.Fatalf("Unexpected dnsPolicy: %v", diff)
								}
							}
						}
					}

					for expName, d := range test.expHorizontalPodAutoscaler {
						for _, u := range manifest.Resources() {
							if u.GetKind() == "HorizontalPodAutoscaler" && u.GetName() == expName {
								got := &v2.HorizontalPodAutoscaler{}
								if err := scheme.Scheme.Convert(&u, got, nil); err != nil {
									t.Fatalf("Failed to convert unstructured to deployment: %v", err)
								}

								minReplicas := int32(0)
								if got.Spec.MinReplicas != nil {
									minReplicas = *got.Spec.MinReplicas
								}
								if diff := cmp.Diff(minReplicas, d.expMinReplicas); diff != "" {
									t.Fatalf("Unexpected minReplicas: %v", diff)
								}

								if diff := cmp.Diff(got.Spec.MaxReplicas, d.expMaxReplicas); diff != "" {
									t.Fatalf("Unexpected maxReplicas: %v", diff)
								}
							}
						}
					}
				})
			}
		})
	}
}

func TestStatefulSetTransform(t *testing.T) {
	tests := []struct {
		DeployName string
		Input      servingv1beta1.KnativeServing
		Expected   map[string]corev1.ResourceRequirements
	}{{
		DeployName: "kafka-source-dispatcher",
		Input: servingv1beta1.KnativeServing{
			ObjectMeta: metav1.ObjectMeta{
				Name: "specific-container-for-deployment",
			},
			Spec: servingv1beta1.KnativeServingSpec{
				CommonSpec: base.CommonSpec{
					Version: test.OperatorFlags.PreviousEventingVersion,
					Workloads: []base.WorkloadOverride{
						{
							Name:     "kafka-source-dispatcher",
							Replicas: ptr.Int32(3),
							Resources: []base.ResourceRequirementsOverride{{
								Container: "kafka-source-dispatcher",
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
		Expected: map[string]corev1.ResourceRequirements{"kafka-source-dispatcher": {
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
			actual, err := manifest.Transform(OverridesTransform(test.Input.GetSpec().GetWorkloadOverrides(), log))
			if err != nil {
				t.Fatalf("Failed to transform manifest: %v", err)
			}
			resources := actual.Filter(mf.ByKind("StatefulSet")).Filter(mf.ByName(test.DeployName)).Resources()
			util.AssertEqual(t, len(resources), 1)
			ss := &appsv1.StatefulSet{}
			if err = scheme.Scheme.Convert(&resources[0], ss, nil); err != nil {
				t.Fatalf("Failed to convert unstructured to deployment: %v", err)
			}
			containers := ss.Spec.Template.Spec.Containers
			for i := range containers {
				expected := test.Expected[containers[i].Name]
				if !reflect.DeepEqual(containers[i].Resources, expected) {
					t.Errorf("\n    Name: %s\n  Expect: %v\n  Actual: %v", containers[i].Name, expected, containers[i].Resources)
				}
			}
		})
	}
}

func TestJobOverridesTransform(t *testing.T) {
	tests := []struct {
		Input    servingv1beta1.KnativeServing
		Expected expJobs
	}{{
		Input: servingv1beta1.KnativeServing{
			ObjectMeta: metav1.ObjectMeta{
				Name: "job-NodeSelector-Tolerations",
			},
			Spec: servingv1beta1.KnativeServingSpec{
				CommonSpec: base.CommonSpec{
					Version: test.OperatorFlags.PreviousEventingVersion,
					Workloads: []base.WorkloadOverride{
						{
							Name:         "storage-version-migration-serving-",
							NodeSelector: map[string]string{"env": "dev"},
							Tolerations: []corev1.Toleration{{
								Key:      corev1.TaintNodeNotReady,
								Operator: corev1.TolerationOpExists,
								Effect:   corev1.TaintEffectNoSchedule,
							}},
						},
					},
				},
			},
		},
		Expected: expJobs{
			expNodeSelector: map[string]string{"env": "dev"},
			expTolerations: []corev1.Toleration{{
				Key:      corev1.TaintNodeNotReady,
				Operator: corev1.TolerationOpExists,
				Effect:   corev1.TaintEffectNoSchedule,
			}},
		},
	}}

	for _, test := range tests {
		t.Run(test.Input.Name, func(t *testing.T) {
			manifest, err := mf.NewManifest("testdata/manifest.yaml")
			if err != nil {
				t.Fatalf("Failed to create manifest: %v", err)
			}
			actual, err := manifest.Transform(OverridesTransform(test.Input.GetSpec().GetWorkloadOverrides(), log))
			if err != nil {
				t.Fatalf("Failed to transform manifest: %v", err)
			}
			resources := actual.Filter(mf.ByKind("Job")).Resources()
			util.AssertEqual(t, len(resources), 1)
			job := &batchv1.Job{}
			if err = scheme.Scheme.Convert(&resources[0], job, nil); err != nil {
				t.Fatalf("Failed to convert unstructured to deployment: %v", err)
			}
			util.AssertDeepEqual(t, job.Spec.Template.Spec.Tolerations, test.Expected.expTolerations)
			util.AssertDeepEqual(t, job.Spec.Template.Spec.NodeSelector, test.Expected.expNodeSelector)
		})
	}
}

func TestDeploymentResourceRequirementsTransform(t *testing.T) {
	tests := []struct {
		DeployName string
		Input      servingv1beta1.KnativeServing
		Expected   map[string]corev1.ResourceRequirements
	}{{
		DeployName: "net-istio-controller",
		Input: servingv1beta1.KnativeServing{
			ObjectMeta: metav1.ObjectMeta{
				Name: "specific-container-for-deployment",
			},
			Spec: servingv1beta1.KnativeServingSpec{
				CommonSpec: base.CommonSpec{
					Version: test.OperatorFlags.PreviousEventingVersion,
					DeploymentOverride: []base.WorkloadOverride{
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
		Expected: map[string]corev1.ResourceRequirements{"controller": {
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
			actual, err := manifest.Transform(OverridesTransform(test.Input.GetSpec().GetWorkloadOverrides(), log))
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

func getEnv(containers []corev1.Container, container string) []corev1.EnvVar {
	for _, c := range containers {
		if c.Name == container {
			return c.Env
		}
	}
	return nil
}

func getProbes(containers []corev1.Container, container string) (readiness *corev1.Probe, liveness *corev1.Probe) {
	for _, c := range containers {
		if c.Name == container {
			return c.ReadinessProbe, c.LivenessProbe
		}
	}
	return nil, nil
}
