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
	appsv1 "k8s.io/api/apps/v1"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes/scheme"

	"knative.dev/operator/pkg/apis/operator/v1alpha1"
	util "knative.dev/operator/pkg/reconciler/common/testing"
)

func TestSinkBindingSelectionModeTransform(t *testing.T) {
	tests := []struct {
		name                     string
		deployment               appsv1.Deployment
		sinkBindingSelectionMode string
		expected                 appsv1.Deployment
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
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unstructuredDeployment := util.MakeUnstructured(t, &tt.deployment)
			instance := &v1alpha1.KnativeEventing{
				Spec: v1alpha1.KnativeEventingSpec{
					SinkBindingSelectionMode: tt.sinkBindingSelectionMode,
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
