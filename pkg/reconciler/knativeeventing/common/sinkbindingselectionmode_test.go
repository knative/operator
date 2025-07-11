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
	"errors"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
	"knative.dev/operator/pkg/apis/operator/base"
	eventingv1beta1 "knative.dev/operator/pkg/apis/operator/v1beta1"
	util "knative.dev/operator/pkg/reconciler/common/testing"
)

func TestSinkBindingSelectionModeTransform(t *testing.T) {
	tests := []struct {
		name                     string
		deployment               appsv1.Deployment
		sinkBindingSelectionMode string
		expected                 appsv1.Deployment
		workloads                []base.WorkloadOverride
	}{{
		name: "UsesDefaultWhenNotSpecified",
		deployment: makeDeployment("eventing-webhook", []corev1.Container{{
			Name: "foo",
			Env: []corev1.EnvVar{{
				Name:  "SINK_BINDING_SELECTION_MODE",
				Value: "toBeOverridden",
			}},
		}}),
		sinkBindingSelectionMode: "",
		expected: makeDeployment("eventing-webhook", []corev1.Container{{
			Name: "foo",
			Env: []corev1.EnvVar{{
				Name:  "SINK_BINDING_SELECTION_MODE",
				Value: "exclusion",
			}},
		}}),
	}, {
		name: "UsesTheSpecifiedValueWhenSpecified",
		deployment: makeDeployment("eventing-webhook", []corev1.Container{{
			Name: "foo",
			Env: []corev1.EnvVar{{
				Name:  "SINK_BINDING_SELECTION_MODE",
				Value: "toBeOverridden",
			}},
		}}),
		sinkBindingSelectionMode: "inclusion",
		expected: makeDeployment("eventing-webhook", []corev1.Container{{
			Name: "foo",
			Env: []corev1.EnvVar{{
				Name:  "SINK_BINDING_SELECTION_MODE",
				Value: "inclusion",
			}},
		}}),
	}, {
		name: "DoesNotTouchOtherDeployments",
		deployment: makeDeployment("some-other-webhook", []corev1.Container{{
			Name: "foo",
			Env: []corev1.EnvVar{{
				Name:  "SINK_BINDING_SELECTION_MODE",
				Value: "notToBeOverridden",
			}},
		}}),
		sinkBindingSelectionMode: "inclusion",
		expected: makeDeployment("eventing-webhook", []corev1.Container{{
			Name: "foo",
			Env: []corev1.EnvVar{{
				Name:  "SINK_BINDING_SELECTION_MODE",
				Value: "notToBeOverridden",
			}},
		}}),
	}, {
		name: "CreatesTheEnvVarIfMissing",
		deployment: makeDeployment("eventing-webhook", []corev1.Container{{
			Name: "foo",
			Env:  []corev1.EnvVar{},
		}}),
		sinkBindingSelectionMode: "inclusion",
		expected: makeDeployment("eventing-webhook", []corev1.Container{{
			Name: "foo",
			Env: []corev1.EnvVar{{
				Name:  "SINK_BINDING_SELECTION_MODE",
				Value: "inclusion",
			}},
		}}),
	}, {
		name: "UpdatesAllContainers",
		deployment: makeDeployment("eventing-webhook", []corev1.Container{
			{
				Name: "container1",
				Env:  []corev1.EnvVar{},
			}, {
				Name: "container2",
				Env:  []corev1.EnvVar{},
			},
		}),
		sinkBindingSelectionMode: "inclusion",
		expected: makeDeployment("eventing-webhook", []corev1.Container{
			{
				Name: "container1",
				Env: []corev1.EnvVar{{
					Name:  "SINK_BINDING_SELECTION_MODE",
					Value: "inclusion",
				}},
			}, {
				Name: "container2",
				Env: []corev1.EnvVar{{
					Name:  "SINK_BINDING_SELECTION_MODE",
					Value: "inclusion",
				}},
			},
		}),
	}, {
		name: "TakesWorkloadOverridesIntoAccount",
		deployment: makeDeployment("eventing-webhook", []corev1.Container{{
			Name: "foo",
			Env: []corev1.EnvVar{{
				Name:  "SINK_BINDING_SELECTION_MODE",
				Value: "toBeOverridden",
			}},
		}}),
		sinkBindingSelectionMode: "",
		workloads: []base.WorkloadOverride{{
			Name: "eventing-webhook",
			Env: []base.EnvRequirementsOverride{{
				Container: "eventing-webhook",
				EnvVars: []corev1.EnvVar{{
					Name:  "SINK_BINDING_SELECTION_MODE",
					Value: "inclusion",
				}},
			}},
		},
		},
		expected: makeDeployment("eventing-webhook", []corev1.Container{{
			Name: "foo",
			Env: []corev1.EnvVar{{
				Name:  "SINK_BINDING_SELECTION_MODE",
				Value: "inclusion",
			}},
		}}),
	}, {
		name: "sinkBindingSelectionModeHasPriorityOverWorkloadOverrides",
		deployment: makeDeployment("eventing-webhook", []corev1.Container{{
			Name: "foo",
			Env: []corev1.EnvVar{{
				Name:  "SINK_BINDING_SELECTION_MODE",
				Value: "toBeOverridden",
			}},
		}}),
		sinkBindingSelectionMode: "smFromSelectionMode",
		workloads: []base.WorkloadOverride{{
			Name: "eventing-webhook",
			Env: []base.EnvRequirementsOverride{{
				Container: "eventing-webhook",
				EnvVars: []corev1.EnvVar{{
					Name:  "SINK_BINDING_SELECTION_MODE",
					Value: "smFromWorkloadOverride",
				}},
			}},
		},
		},
		expected: makeDeployment("eventing-webhook", []corev1.Container{{
			Name: "foo",
			Env: []corev1.EnvVar{{
				Name:  "SINK_BINDING_SELECTION_MODE",
				Value: "smFromSelectionMode",
			}},
		}}),
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unstructuredDeployment := util.MakeUnstructured(t, &tt.deployment)
			instance := &eventingv1beta1.KnativeEventing{
				Spec: eventingv1beta1.KnativeEventingSpec{
					SinkBindingSelectionMode: tt.sinkBindingSelectionMode,
					CommonSpec: base.CommonSpec{
						Workloads: tt.workloads,
					},
				},
			}
			transform := SinkBindingSelectionModeTransform(instance, log)
			transform(&unstructuredDeployment)

			var deployment = &appsv1.Deployment{}
			err := scheme.Scheme.Convert(&unstructuredDeployment, deployment, nil)
			util.AssertEqual(t, err, nil)
			util.AssertDeepEqual(t, deployment.Spec, tt.expected.Spec)
		})
	}
}

func makeDeployment(name string, containers []corev1.Container) appsv1.Deployment {
	return appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: containers,
				},
			},
		},
	}
}

func Test_sinkBindingSelectionModeFromWorkloadOverrides(t *testing.T) {
	tests := []struct {
		name         string
		instanceSpec *eventingv1beta1.KnativeEventingSpec
		want         string
	}{
		{
			name: "should_return_SBSM_from_webhook",
			instanceSpec: &eventingv1beta1.KnativeEventingSpec{
				CommonSpec: base.CommonSpec{
					Workloads: []base.WorkloadOverride{{
						Name: "eventing-webhook",
						Env: []base.EnvRequirementsOverride{{
							Container: "eventing-webhook",
							EnvVars: []corev1.EnvVar{{
								Name:  "SINK_BINDING_SELECTION_MODE",
								Value: "sbsmFromWorkloadOverride",
							}},
						}},
					},
					},
				},
			},
			want: "sbsmFromWorkloadOverride",
		},
		{
			name: "should_return_SBSM_from_webhook_with_multiple_workloadOverrides",
			instanceSpec: &eventingv1beta1.KnativeEventingSpec{
				CommonSpec: base.CommonSpec{
					Workloads: []base.WorkloadOverride{{
						Name: "another-workload",
						Env: []base.EnvRequirementsOverride{{
							Container: "eventing-webhook",
							EnvVars: []corev1.EnvVar{{
								Name:  "SINK_BINDING_SELECTION_MODE",
								Value: "anotherSbsmFromWorkloadOverride",
							}},
						}}}, {
						Name: "eventing-webhook",
						Env: []base.EnvRequirementsOverride{{
							Container: "eventing-webhook",
							EnvVars: []corev1.EnvVar{{
								Name:  "SINK_BINDING_SELECTION_MODE",
								Value: "sbsmFromWorkloadOverride",
							}},
						}}},
					},
				},
			},
			want: "sbsmFromWorkloadOverride",
		},
		{
			name: "should_return_SBSM_from_webhook_with_multiple_containers",
			instanceSpec: &eventingv1beta1.KnativeEventingSpec{
				CommonSpec: base.CommonSpec{
					Workloads: []base.WorkloadOverride{{
						Name: "eventing-webhook",
						Env: []base.EnvRequirementsOverride{{
							Container: "another-container",
							EnvVars: []corev1.EnvVar{{
								Name:  "SINK_BINDING_SELECTION_MODE",
								Value: "anotherSbsmFromWorkloadOverride",
							}}}, {
							Container: "eventing-webhook",
							EnvVars: []corev1.EnvVar{{
								Name:  "SINK_BINDING_SELECTION_MODE",
								Value: "sbsmFromWorkloadOverride",
							}},
						}}},
					},
				},
			},
			want: "sbsmFromWorkloadOverride",
		},
		{
			name: "should_take_deprecated_deployment_overrides_into_account_too",
			instanceSpec: &eventingv1beta1.KnativeEventingSpec{
				CommonSpec: base.CommonSpec{
					DeploymentOverride: []base.WorkloadOverride{{
						Name: "eventing-webhook",
						Env: []base.EnvRequirementsOverride{{
							Container: "eventing-webhook",
							EnvVars: []corev1.EnvVar{{
								Name:  "SINK_BINDING_SELECTION_MODE",
								Value: "sbsmFromDeploymentOverride",
							}},
						}}},
					},
				},
			},
			want: "sbsmFromDeploymentOverride",
		},
		{
			name: "workload_overrides_should_have_priority_over_deprecated_deployment_overrides",
			instanceSpec: &eventingv1beta1.KnativeEventingSpec{
				CommonSpec: base.CommonSpec{
					Workloads: []base.WorkloadOverride{{
						Name: "eventing-webhook",
						Env: []base.EnvRequirementsOverride{{
							Container: "eventing-webhook",
							EnvVars: []corev1.EnvVar{{
								Name:  "SINK_BINDING_SELECTION_MODE",
								Value: "sbsmFromWorkloadOverride",
							}},
						}}},
					},
					DeploymentOverride: []base.WorkloadOverride{{
						Name: "eventing-webhook",
						Env: []base.EnvRequirementsOverride{{
							Container: "eventing-webhook",
							EnvVars: []corev1.EnvVar{{
								Name:  "SINK_BINDING_SELECTION_MODE",
								Value: "sbsmFromDeploymentOverride",
							}},
						}}},
					},
				},
			},
			want: "sbsmFromWorkloadOverride",
		},
		{
			name: "should_return_empty_string_if_not_found",
			instanceSpec: &eventingv1beta1.KnativeEventingSpec{
				CommonSpec: base.CommonSpec{
					Workloads: []base.WorkloadOverride{{
						Name: "eventing-webhook",
						Env: []base.EnvRequirementsOverride{{
							Container: "eventing-webhook",
							EnvVars: []corev1.EnvVar{{
								Name:  "ANOTHER_ENV",
								Value: "foobar",
							}},
						}}},
					},
				},
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := &eventingv1beta1.KnativeEventing{
				Spec: *tt.instanceSpec,
			}

			if got := sinkBindingSelectionModeFromWorkloadOverrides(instance); got != tt.want {
				t.Errorf("sinkBindingSelectionModeFromWorkloadOverrides() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestSinkBindingSelectionModeTransformErrorHandling tests error scenarios
func TestSinkBindingSelectionModeTransformErrorHandling(t *testing.T) {
	tests := []struct {
		name         string
		unstructured *unstructured.Unstructured
		instance     *eventingv1beta1.KnativeEventing
		expectError  bool
	}{
		{
			name: "should_ignore_non_deployment_resources",
			unstructured: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "ConfigMap",
					"metadata": map[string]interface{}{
						"name": "eventing-webhook",
					},
				},
			},
			instance: &eventingv1beta1.KnativeEventing{
				Spec: eventingv1beta1.KnativeEventingSpec{
					SinkBindingSelectionMode: "inclusion",
				},
			},
			expectError: false,
		},
		{
			name: "should_ignore_deployments_with_different_name",
			unstructured: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "apps/v1",
					"kind":       "Deployment",
					"metadata": map[string]interface{}{
						"name": "different-webhook",
					},
					"spec": map[string]interface{}{
						"template": map[string]interface{}{
							"spec": map[string]interface{}{
								"containers": []interface{}{
									map[string]interface{}{
										"name": "container",
										"env":  []interface{}{},
									},
								},
							},
						},
					},
				},
			},
			instance: &eventingv1beta1.KnativeEventing{
				Spec: eventingv1beta1.KnativeEventingSpec{
					SinkBindingSelectionMode: "inclusion",
				},
			},
			expectError: false,
		},
		{
			name: "should_handle_invalid_deployment_spec",
			unstructured: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "apps/v1",
					"kind":       "Deployment",
					"metadata": map[string]interface{}{
						"name": "eventing-webhook",
					},
					"spec": map[string]interface{}{
						"template": map[string]interface{}{
							"spec": map[string]interface{}{
								"containers": "invalid_type", // Should be array
							},
						},
					},
				},
			},
			instance: &eventingv1beta1.KnativeEventing{
				Spec: eventingv1beta1.KnativeEventingSpec{
					SinkBindingSelectionMode: "inclusion",
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transform := SinkBindingSelectionModeTransform(tt.instance, log)
			err := transform(tt.unstructured)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

// TestSinkBindingSelectionModeTransformEdgeCases tests edge cases and boundary conditions
func TestSinkBindingSelectionModeTransformEdgeCases(t *testing.T) {
	tests := []struct {
		name                     string
		deployment               appsv1.Deployment
		sinkBindingSelectionMode string
		expected                 appsv1.Deployment
		workloads                []base.WorkloadOverride
	}{
		{
			name:                     "should_handle_empty_containers_list",
			deployment:               makeDeployment("eventing-webhook", []corev1.Container{}),
			sinkBindingSelectionMode: "inclusion",
			expected:                 makeDeployment("eventing-webhook", []corev1.Container{}),
		},
		{
			name: "should_handle_container_with_nil_env_vars",
			deployment: makeDeployment("eventing-webhook", []corev1.Container{{
				Name: "container",
				Env:  nil,
			}}),
			sinkBindingSelectionMode: "inclusion",
			expected: makeDeployment("eventing-webhook", []corev1.Container{{
				Name: "container",
				Env: []corev1.EnvVar{{
					Name:  "SINK_BINDING_SELECTION_MODE",
					Value: "inclusion",
				}},
			}}),
		},
		{
			name: "should_handle_multiple_env_vars_with_same_name",
			deployment: makeDeployment("eventing-webhook", []corev1.Container{{
				Name: "container",
				Env: []corev1.EnvVar{
					{Name: "OTHER_ENV", Value: "other_value"},
					{Name: "SINK_BINDING_SELECTION_MODE", Value: "old_value"},
					{Name: "SINK_BINDING_SELECTION_MODE", Value: "duplicate_value"},
				},
			}}),
			sinkBindingSelectionMode: "new_value",
			expected: makeDeployment("eventing-webhook", []corev1.Container{{
				Name: "container",
				Env: []corev1.EnvVar{
					{Name: "OTHER_ENV", Value: "other_value"},
					{Name: "SINK_BINDING_SELECTION_MODE", Value: "new_value"},
					{Name: "SINK_BINDING_SELECTION_MODE", Value: "duplicate_value"},
				},
			}}),
		},
		{
			name: "should_handle_workload_override_with_empty_env_vars",
			deployment: makeDeployment("eventing-webhook", []corev1.Container{{
				Name: "container",
				Env:  []corev1.EnvVar{},
			}}),
			sinkBindingSelectionMode: "",
			workloads: []base.WorkloadOverride{{
				Name: "eventing-webhook",
				Env: []base.EnvRequirementsOverride{{
					Container: "eventing-webhook",
					EnvVars:   []corev1.EnvVar{}, // Empty env vars
				}},
			}},
			expected: makeDeployment("eventing-webhook", []corev1.Container{{
				Name: "container",
				Env: []corev1.EnvVar{{
					Name:  "SINK_BINDING_SELECTION_MODE",
					Value: "exclusion", // Should use default
				}},
			}}),
		},
		{
			name: "should_handle_workload_override_with_wrong_container_name",
			deployment: makeDeployment("eventing-webhook", []corev1.Container{{
				Name: "container",
				Env:  []corev1.EnvVar{},
			}}),
			sinkBindingSelectionMode: "",
			workloads: []base.WorkloadOverride{{
				Name: "eventing-webhook",
				Env: []base.EnvRequirementsOverride{{
					Container: "wrong-container-name",
					EnvVars: []corev1.EnvVar{{
						Name:  "SINK_BINDING_SELECTION_MODE",
						Value: "from_workload_override",
					}},
				}},
			}},
			expected: makeDeployment("eventing-webhook", []corev1.Container{{
				Name: "container",
				Env: []corev1.EnvVar{{
					Name:  "SINK_BINDING_SELECTION_MODE",
					Value: "exclusion", // Should use default since container name doesn't match
				}},
			}}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unstructuredDeployment := util.MakeUnstructured(t, &tt.deployment)
			instance := &eventingv1beta1.KnativeEventing{
				Spec: eventingv1beta1.KnativeEventingSpec{
					SinkBindingSelectionMode: tt.sinkBindingSelectionMode,
					CommonSpec: base.CommonSpec{
						Workloads: tt.workloads,
					},
				},
			}
			transform := SinkBindingSelectionModeTransform(instance, log)
			err := transform(&unstructuredDeployment)
			if err != nil {
				t.Errorf("Transform failed: %v", err)
				return
			}

			var deployment = &appsv1.Deployment{}
			err = scheme.Scheme.Convert(&unstructuredDeployment, deployment, nil)
			util.AssertEqual(t, err, nil)
			util.AssertDeepEqual(t, deployment.Spec, tt.expected.Spec)
		})
	}
}

// TestSinkBindingSelectionModeTransformLogging tests logging behavior
func TestSinkBindingSelectionModeTransformLogging(t *testing.T) {
	// Test successful transformation with logging
	deployment := makeDeployment("eventing-webhook", []corev1.Container{{
		Name: "container",
		Env:  []corev1.EnvVar{},
	}})
	unstructuredDeployment := util.MakeUnstructured(t, &deployment)
	instance := &eventingv1beta1.KnativeEventing{
		Spec: eventingv1beta1.KnativeEventingSpec{
			SinkBindingSelectionMode: "inclusion",
		},
	}

	transform := SinkBindingSelectionModeTransform(instance, log)
	err := transform(&unstructuredDeployment)
	if err != nil {
		t.Errorf("Transform failed: %v", err)
	}

	// Test error logging with malformed object that should cause conversion error
	malformedUnstructured := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]interface{}{
				"name": "eventing-webhook",
			},
			"spec": map[string]interface{}{
				"template": map[string]interface{}{
					"spec": map[string]interface{}{
						"containers": "invalid_type", // This should cause conversion error
					},
				},
			},
		},
	}

	transform = SinkBindingSelectionModeTransform(instance, log)
	err = transform(malformedUnstructured)
	if err == nil {
		t.Error("Expected error but got none")
	}
}

// TestSinkBindingSelectionModeTransformConversionErrors tests conversion error scenarios
func TestSinkBindingSelectionModeTransformConversionErrors(t *testing.T) {
	tests := []struct {
		name         string
		unstructured *unstructured.Unstructured
		instance     *eventingv1beta1.KnativeEventing
		expectError  bool
	}{
		{
			name: "should_handle_conversion_error_on_invalid_spec",
			unstructured: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "apps/v1",
					"kind":       "Deployment",
					"metadata": map[string]interface{}{
						"name": "eventing-webhook",
					},
					"spec": map[string]interface{}{
						"template": map[string]interface{}{
							"spec": map[string]interface{}{
								"containers": "invalid_type", // Should be array
							},
						},
					},
				},
			},
			instance: &eventingv1beta1.KnativeEventing{
				Spec: eventingv1beta1.KnativeEventingSpec{
					SinkBindingSelectionMode: "inclusion",
				},
			},
			expectError: true,
		},
		{
			name: "should_handle_conversion_error_on_invalid_template_spec",
			unstructured: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "apps/v1",
					"kind":       "Deployment",
					"metadata": map[string]interface{}{
						"name": "eventing-webhook",
					},
					"spec": map[string]interface{}{
						"template": "invalid_template_type", // Should be object
					},
				},
			},
			instance: &eventingv1beta1.KnativeEventing{
				Spec: eventingv1beta1.KnativeEventingSpec{
					SinkBindingSelectionMode: "inclusion",
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transform := SinkBindingSelectionModeTransform(tt.instance, log)
			err := transform(tt.unstructured)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

// TestSinkBindingSelectionModeTransformResourceFiltering tests resource filtering behavior
func TestSinkBindingSelectionModeTransformResourceFiltering(t *testing.T) {
	tests := []struct {
		name         string
		unstructured *unstructured.Unstructured
		instance     *eventingv1beta1.KnativeEventing
		expectError  bool
	}{
		{
			name: "should_ignore_configmap_resources",
			unstructured: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "ConfigMap",
					"metadata": map[string]interface{}{
						"name": "eventing-webhook",
					},
					"data": map[string]interface{}{
						"key": "value",
					},
				},
			},
			instance: &eventingv1beta1.KnativeEventing{
				Spec: eventingv1beta1.KnativeEventingSpec{
					SinkBindingSelectionMode: "inclusion",
				},
			},
			expectError: false,
		},
		{
			name: "should_ignore_service_resources",
			unstructured: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Service",
					"metadata": map[string]interface{}{
						"name": "eventing-webhook",
					},
					"spec": map[string]interface{}{
						"ports": []interface{}{},
					},
				},
			},
			instance: &eventingv1beta1.KnativeEventing{
				Spec: eventingv1beta1.KnativeEventingSpec{
					SinkBindingSelectionMode: "inclusion",
				},
			},
			expectError: false,
		},
		{
			name: "should_ignore_deployments_with_different_name",
			unstructured: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "apps/v1",
					"kind":       "Deployment",
					"metadata": map[string]interface{}{
						"name": "different-webhook",
					},
					"spec": map[string]interface{}{
						"template": map[string]interface{}{
							"spec": map[string]interface{}{
								"containers": []interface{}{
									map[string]interface{}{
										"name": "container",
										"env":  []interface{}{},
									},
								},
							},
						},
					},
				},
			},
			instance: &eventingv1beta1.KnativeEventing{
				Spec: eventingv1beta1.KnativeEventingSpec{
					SinkBindingSelectionMode: "inclusion",
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transform := SinkBindingSelectionModeTransform(tt.instance, log)
			err := transform(tt.unstructured)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

// TestSinkBindingSelectionModeTransform_SecondConvertError simulates a failure in the second conversion (deployment -> unstructured)
func TestSinkBindingSelectionModeTransform_SecondConvertError(t *testing.T) {
	deployment := makeDeployment("eventing-webhook", []corev1.Container{{
		Name: "container",
		Env:  []corev1.EnvVar{},
	}})
	unstructuredDeployment := util.MakeUnstructured(t, &deployment)
	instance := &eventingv1beta1.KnativeEventing{
		Spec: eventingv1beta1.KnativeEventingSpec{
			SinkBindingSelectionMode: "inclusion",
		},
	}

	callCount := 0
	convert := func(in, out, context interface{}) error {
		callCount++
		if callCount == 2 {
			return errors.New("forced error for test")
		}
		return scheme.Scheme.Convert(in, out, context)
	}

	transform := SinkBindingSelectionModeTransform(instance, log, convert)
	err := transform(&unstructuredDeployment)
	if err == nil {
		t.Error("Expected error in second conversion but got none")
	}
}
