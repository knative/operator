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

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	eventingv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
)

type updateDeploymentImageTest struct {
	name       string
	containers []corev1.Container
	registry   eventingv1alpha1.Registry
	expected   []corev1.Container
}

var updateDeploymentImageTests = []updateDeploymentImageTest{
	{
		name: "UsesNameFromDefault",
		containers: []corev1.Container{{
			Name:  "queue",
			Image: "gcr.io/knative-releases/github.com/knative/eventing/cmd/queue@sha256:1e40c99ff5977daa2d69873fff604c6d09651af1f9ff15aadf8849b3ee77ab45"},
		},
		registry: eventingv1alpha1.Registry{
			Default: "new-registry.io/test/path/${NAME}:new-tag",
		},
		expected: []corev1.Container{{
			Name:  "queue",
			Image: "new-registry.io/test/path/queue:new-tag"},
		},
	},
	{
		name: "UsesContainerNamePerContainer",
		containers: []corev1.Container{
			{
				Name:  "container1",
				Image: "gcr.io/cmd/queue:test",
			},
			{
				Name:  "container2",
				Image: "gcr.io/cmd/queue:test",
			},
		},
		registry: eventingv1alpha1.Registry{
			Override: map[string]string{
				"container1": "new-registry.io/test/path/new-container-1:new-tag",
				"container2": "new-registry.io/test/path/new-container-2:new-tag",
			},
		},
		expected: []corev1.Container{
			{
				Name:  "container1",
				Image: "new-registry.io/test/path/new-container-1:new-tag",
			},
			{
				Name:  "container2",
				Image: "new-registry.io/test/path/new-container-2:new-tag",
			},
		},
	},
	{
		name: "UsesOverrideFromDefault",
		containers: []corev1.Container{{
			Name:  "queue",
			Image: "gcr.io/knative-releases/github.com/knative/eventing/cmd/queue@sha256:1e40c99ff5977daa2d69873fff604c6d09651af1f9ff15aadf8849b3ee77ab45"},
		},
		registry: eventingv1alpha1.Registry{
			Default: "new-registry.io/test/path/${NAME}:new-tag",
			Override: map[string]string{
				"queue": "new-registry.io/test/path/new-value:new-override-tag",
			},
		},
		expected: []corev1.Container{{
			Name:  "queue",
			Image: "new-registry.io/test/path/new-value:new-override-tag"},
		},
	},
	{
		name: "NoChangeOverrideWithDifferentName",
		containers: []corev1.Container{{
			Name:  "image",
			Image: "docker.io/name/image:tag2"},
		},
		registry: eventingv1alpha1.Registry{
			Override: map[string]string{
				"Unused": "new-registry.io/test/path",
			},
		},
		expected: []corev1.Container{{
			Name:  "image",
			Image: "docker.io/name/image:tag2"},
		},
	},
	{
		name: "NoChange",
		containers: []corev1.Container{{
			Name:  "queue",
			Image: "gcr.io/knative-releases/github.com/knative/eventing/cmd/queue@sha256:1e40c99ff5977daa2d69873fff604c6d09651af1f9ff15aadf8849b3ee77ab45"},
		},
		registry: eventingv1alpha1.Registry{},
		expected: []corev1.Container{{
			Name:  "queue",
			Image: "gcr.io/knative-releases/github.com/knative/eventing/cmd/queue@sha256:1e40c99ff5977daa2d69873fff604c6d09651af1f9ff15aadf8849b3ee77ab45"},
		},
	},
	{
		name: "OverrideEnvVarImage",
		containers: []corev1.Container{{
			Env: []corev1.EnvVar{{Name: "SOME_IMAGE", Value: "gcr.io/foo/bar"}},
		}},
		registry: eventingv1alpha1.Registry{
			Override: map[string]string{
				"SOME_IMAGE": "docker.io/my/overridden-image",
			},
		},
		expected: []corev1.Container{{
			Env: []corev1.EnvVar{{Name: "SOME_IMAGE", Value: "docker.io/my/overridden-image"}},
		}},
	},
	{
		name: "NoOverrideEnvVarImage",
		containers: []corev1.Container{{
			Env: []corev1.EnvVar{{Name: "SOME_IMAGE", Value: "gcr.io/foo/bar"}},
		}},
		registry: eventingv1alpha1.Registry{
			Override: map[string]string{
				"OTHER_IMAGE": "docker.io/my/overridden-image",
			},
		},
		expected: []corev1.Container{{
			Env: []corev1.EnvVar{{Name: "SOME_IMAGE", Value: "gcr.io/foo/bar"}},
		}},
	},
	{
		name: "NoOverrideEnvVarImageAndContainerImageBoth",
		containers: []corev1.Container{{
			Name:  "queue",
			Image: "gcr.io/knative-releases/github.com/knative/eventing/cmd/queue@sha256:1e40c99ff5977daa2d69873fff604c6d09651af1f9ff15aadf8849b3ee77ab45",
			Env:   []corev1.EnvVar{{Name: "SOME_IMAGE", Value: "gcr.io/foo/bar"}},
		}},
		registry: eventingv1alpha1.Registry{
			Override: map[string]string{
				"queue":      "new-registry.io/test/path/new-value:new-override-tag",
				"SOME_IMAGE": "docker.io/my/overridden-image",
			},
		},
		expected: []corev1.Container{{
			Name:  "queue",
			Image: "new-registry.io/test/path/new-value:new-override-tag",
			Env:   []corev1.EnvVar{{Name: "SOME_IMAGE", Value: "docker.io/my/overridden-image"}},
		}},
	},
	{
		name: "OverrideWithDeploymentContainer",
		containers: []corev1.Container{
			{
				Name:  "container1",
				Image: "gcr.io/cmd/queue:test",
			},
			{
				Name:  "container2",
				Image: "gcr.io/cmd/queue:test",
			},
		},
		registry: eventingv1alpha1.Registry{
			Override: map[string]string{
				"container1": "new-registry.io/test/path/new-container-1:new-tag",
				"container2": "new-registry.io/test/path/new-container-2:new-tag",
				"OverrideWithDeploymentContainer/container1": "new-registry.io/test/path/OverrideWithDeploymentContainer/container-1:new-tag",
				"OverrideWithDeploymentContainer/container2": "new-registry.io/test/path/OverrideWithDeploymentContainer/container-2:new-tag",
			},
		},
		expected: []corev1.Container{
			{
				Name:  "container1",
				Image: "new-registry.io/test/path/OverrideWithDeploymentContainer/container-1:new-tag",
			},
			{
				Name:  "container2",
				Image: "new-registry.io/test/path/OverrideWithDeploymentContainer/container-2:new-tag",
			},
		},
	},
	{
		name: "OverridePartialWithDeploymentContainer",
		containers: []corev1.Container{
			{
				Name:  "container1",
				Image: "gcr.io/cmd/queue:test",
			},
			{
				Name:  "container2",
				Image: "gcr.io/cmd/queue:test",
			},
		},
		registry: eventingv1alpha1.Registry{
			Override: map[string]string{
				"container1": "new-registry.io/test/path/new-container-1:new-tag",
				"container2": "new-registry.io/test/path/new-container-2:new-tag",
				"OverridePartialWithDeploymentContainer/container1": "new-registry.io/test/path/OverridePartialWithDeploymentContainer/container-1:new-tag",
			},
		},
		expected: []corev1.Container{
			{
				Name:  "container1",
				Image: "new-registry.io/test/path/OverridePartialWithDeploymentContainer/container-1:new-tag",
			},
			{
				Name:  "container2",
				Image: "new-registry.io/test/path/new-container-2:new-tag",
			},
		},
	},
	{
		name: "OverrideWithDeploymentName",
		containers: []corev1.Container{
			{
				Name:  "container1",
				Image: "gcr.io/cmd/queue:test",
			},
			{
				Name:  "container2",
				Image: "gcr.io/cmd/queue:test",
			},
		},
		registry: eventingv1alpha1.Registry{
			Override: map[string]string{
				"OverrideWithDeploymentName/container1": "new-registry.io/test/path/OverrideWithDeploymentName/container-1:new-tag",
				"OverrideWithDeploymentName/container2": "new-registry.io/test/path/OverrideWithDeploymentName/container-2:new-tag",
			},
		},
		expected: []corev1.Container{
			{
				Name:  "container1",
				Image: "new-registry.io/test/path/OverrideWithDeploymentName/container-1:new-tag",
			},
			{
				Name:  "container2",
				Image: "new-registry.io/test/path/OverrideWithDeploymentName/container-2:new-tag",
			},
		},
	},
}

func TestDeploymentTransform(t *testing.T) {
	for _, tt := range updateDeploymentImageTests {
		t.Run(tt.name, func(t *testing.T) {
			runDeploymentTransformTest(t, &tt)
		})
	}
}

func runDeploymentTransformTest(t *testing.T, tt *updateDeploymentImageTest) {
	unstructuredDeployment := makeUnstructured(t, makeDeployment(t, tt.name, corev1.PodSpec{Containers: tt.containers}))
	instance := &eventingv1alpha1.KnativeEventing{
		Spec: eventingv1alpha1.KnativeEventingSpec{
			Registry: tt.registry,
		},
	}
	deploymentTransform := DeploymentTransform(instance, log)
	deploymentTransform(&unstructuredDeployment)
	validateUnstructedDeploymentChanged(t, tt, &unstructuredDeployment)

	unstructuredJob := makeUnstructured(t, makeJob(t, tt.name, corev1.PodSpec{Containers: tt.containers}))
	instance = &eventingv1alpha1.KnativeEventing{
		Spec: eventingv1alpha1.KnativeEventingSpec{
			Registry: tt.registry,
		},
	}
	jobTransform := DeploymentTransform(instance, log)
	jobTransform(&unstructuredJob)
	validateUnstructedJobChanged(t, tt, &unstructuredJob)
}

func validateUnstructedDeploymentChanged(t *testing.T, tt *updateDeploymentImageTest, u *unstructured.Unstructured) {
	var deployment = &appsv1.Deployment{}
	err := scheme.Scheme.Convert(u, deployment, nil)
	assertEqual(t, err, nil)
	assertDeepEqual(t, deployment.Spec.Template.Spec.Containers, tt.expected)
}

func validateUnstructedJobChanged(t *testing.T, tt *updateDeploymentImageTest, u *unstructured.Unstructured) {
	var job = &batchv1.Job{}
	err := scheme.Scheme.Convert(u, job, nil)
	assertEqual(t, err, nil)
	assertDeepEqual(t, job.Spec.Template.Spec.Containers, tt.expected)
}

func makeDeployment(t *testing.T, name string, podSpec corev1.PodSpec) *appsv1.Deployment {
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind: "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: podSpec,
			},
		},
	}
}

func makeJob(t *testing.T, name string, podSpec corev1.PodSpec) *batchv1.Job {
	return &batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			Kind: "Job",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: podSpec,
			},
		},
	}
}

func makeUnstructured(t *testing.T, obj interface{}) unstructured.Unstructured {
	var result = unstructured.Unstructured{}
	err := scheme.Scheme.Convert(obj, &result, nil)
	if err != nil {
		t.Fatalf("Could not create unstructured object: %v, err: %v", result, err)
	}
	return result
}

type addImagePullSecretsTest struct {
	name            string
	existingSecrets []corev1.LocalObjectReference
	registry        eventingv1alpha1.Registry
	expectedSecrets []corev1.LocalObjectReference
}

var addImagePullSecretsTests = []addImagePullSecretsTest{
	{
		name:            "LeavesSecretsEmptyByDefault",
		existingSecrets: nil,
		registry:        eventingv1alpha1.Registry{},
		expectedSecrets: nil,
	},
	{
		name:            "AddsImagePullSecrets",
		existingSecrets: nil,
		registry: eventingv1alpha1.Registry{
			ImagePullSecrets: []corev1.LocalObjectReference{{Name: "new-secret"}},
		},
		expectedSecrets: []corev1.LocalObjectReference{{Name: "new-secret"}},
	},
	{
		name:            "SupportsMultipleImagePullSecrets",
		existingSecrets: nil,
		registry: eventingv1alpha1.Registry{
			ImagePullSecrets: []corev1.LocalObjectReference{
				{Name: "new-secret-1"},
				{Name: "new-secret-2"},
			},
		},
		expectedSecrets: []corev1.LocalObjectReference{
			{Name: "new-secret-1"},
			{Name: "new-secret-2"},
		},
	},
	{
		name:            "MergesAdditionalSecretsWithAnyPreexisting",
		existingSecrets: []corev1.LocalObjectReference{{Name: "existing-secret"}},
		registry: eventingv1alpha1.Registry{
			ImagePullSecrets: []corev1.LocalObjectReference{
				{Name: "new-secret"},
			},
		},
		expectedSecrets: []corev1.LocalObjectReference{
			{Name: "existing-secret"},
			{Name: "new-secret"},
		},
	},
}

func TestImagePullSecrets(t *testing.T) {
	for _, tt := range addImagePullSecretsTests {
		t.Run(tt.name, func(t *testing.T) {
			runImagePullSecretsTest(t, &tt)
		})
	}
}

func runImagePullSecretsTest(t *testing.T, tt *addImagePullSecretsTest) {
	unstructuredDeployment := makeUnstructured(t, makeDeployment(t, tt.name, corev1.PodSpec{ImagePullSecrets: tt.existingSecrets}))
	instance := &eventingv1alpha1.KnativeEventing{
		Spec: eventingv1alpha1.KnativeEventingSpec{
			Registry: tt.registry,
		},
	}
	deploymentTransform := DeploymentTransform(instance, log)
	deploymentTransform(&unstructuredDeployment)

	var deployment = &appsv1.Deployment{}
	err := scheme.Scheme.Convert(&unstructuredDeployment, deployment, nil)

	assertEqual(t, err, nil)
	assertDeepEqual(t, deployment.Spec.Template.Spec.ImagePullSecrets, tt.expectedSecrets)
}

func assertEqual(t *testing.T, actual, expected interface{}) {
	if actual == expected {
		return
	}
	t.Fatalf("expected does not equal actual. \nExpected: %v\nActual: %v", expected, actual)
}

func assertDeepEqual(t *testing.T, actual, expected interface{}) {
	if reflect.DeepEqual(actual, expected) {
		return
	}
	t.Fatalf("expected does not deep equal actual. \nExpected: %T %+v\nActual:   %T %+v", expected, expected, actual, actual)
}
