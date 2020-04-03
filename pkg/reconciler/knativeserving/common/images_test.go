/*
Copyright 2019 The Knative Authors

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
	caching "knative.dev/caching/pkg/apis/caching/v1alpha1"
	servingv1alpha1 "knative.dev/serving-operator/pkg/apis/serving/v1alpha1"

	appsv1 "k8s.io/api/apps/v1"
)

type updateImageTest struct {
	name       string
	containers []corev1.Container
	registry   servingv1alpha1.Registry
	expected   []string
}

var updateImageTests = []updateImageTest{
	{
		name: "UsesNameFromDefault",
		containers: []corev1.Container{{
			Name:  "queue",
			Image: "gcr.io/knative-releases/github.com/knative/serving/cmd/queue@sha256:1e40c99ff5977daa2d69873fff604c6d09651af1f9ff15aadf8849b3ee77ab45"},
		},
		registry: servingv1alpha1.Registry{
			Default: "new-registry.io/test/path/${NAME}:new-tag",
		},
		expected: []string{"new-registry.io/test/path/queue:new-tag"},
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
		registry: servingv1alpha1.Registry{
			Override: map[string]string{
				"container1": "new-registry.io/test/path/new-container-1:new-tag",
				"container2": "new-registry.io/test/path/new-container-2:new-tag",
			},
		},
		expected: []string{
			"new-registry.io/test/path/new-container-1:new-tag",
			"new-registry.io/test/path/new-container-2:new-tag",
		},
	},
	{
		name: "UsesOverrideFromDefault",
		containers: []corev1.Container{{
			Name:  "queue",
			Image: "gcr.io/knative-releases/github.com/knative/serving/cmd/queue@sha256:1e40c99ff5977daa2d69873fff604c6d09651af1f9ff15aadf8849b3ee77ab45"},
		},
		registry: servingv1alpha1.Registry{
			Default: "new-registry.io/test/path/${NAME}:new-tag",
			Override: map[string]string{
				"queue": "new-registry.io/test/path/new-value:new-override-tag",
			},
		},
		expected: []string{"new-registry.io/test/path/new-value:new-override-tag"},
	},
	{
		name: "NoChangeOverrideWithDifferentName",
		containers: []corev1.Container{{
			Name:  "image",
			Image: "docker.io/name/image:tag2"},
		},
		registry: servingv1alpha1.Registry{
			Override: map[string]string{
				"Unused": "new-registry.io/test/path",
			},
		},
		expected: []string{"docker.io/name/image:tag2"},
	},
	{
		name: "NoChange",
		containers: []corev1.Container{{
			Name:  "queue",
			Image: "gcr.io/knative-releases/github.com/knative/serving/cmd/queue@sha256:1e40c99ff5977daa2d69873fff604c6d09651af1f9ff15aadf8849b3ee77ab45"},
		},
		registry: servingv1alpha1.Registry{},
		expected: []string{"gcr.io/knative-releases/github.com/knative/serving/cmd/queue@sha256:1e40c99ff5977daa2d69873fff604c6d09651af1f9ff15aadf8849b3ee77ab45"},
	},
}

func TestResourceTransform(t *testing.T) {
	for _, tt := range updateImageTests {
		t.Run(tt.name, func(t *testing.T) {
			runResourceTransformTest(t, &tt)
		})
	}
}

func runResourceTransformTest(t *testing.T, tt *updateImageTest) {
	// test for deployment
	unstructuredDeployment := makeUnstructured(t, makeDeployment(tt.name, corev1.PodSpec{Containers: tt.containers}))
	instance := &servingv1alpha1.KnativeServing{
		Spec: servingv1alpha1.KnativeServingSpec{
			Registry: tt.registry,
		},
	}
	deploymentTransform := ImageTransform(instance, log)
	deploymentTransform(&unstructuredDeployment)
	validateUnstructedDeploymentChanged(t, tt, &unstructuredDeployment)

	// test for daemonSet
	unstructuredDaemonSet := makeUnstructured(t, makeDaemonSet(tt.name, corev1.PodSpec{Containers: tt.containers}))
	instance = &servingv1alpha1.KnativeServing{
		Spec: servingv1alpha1.KnativeServingSpec{
			Registry: tt.registry,
		},
	}
	daemonSetTransform := ImageTransform(instance, log)
	daemonSetTransform(&unstructuredDaemonSet)
	validateUnstructedDaemonSetChanged(t, tt, &unstructuredDaemonSet)
}

func validateUnstructedDeploymentChanged(t *testing.T, tt *updateImageTest, u *unstructured.Unstructured) {
	var deployment = &appsv1.Deployment{}
	err := scheme.Scheme.Convert(u, deployment, nil)
	assertEqual(t, err, nil)
	for i, expected := range tt.expected {
		assertEqual(t, deployment.Spec.Template.Spec.Containers[i].Image, expected)
	}
}

func validateUnstructedDaemonSetChanged(t *testing.T, tt *updateImageTest, u *unstructured.Unstructured) {
	var daemonSet = &appsv1.DaemonSet{}
	err := scheme.Scheme.Convert(u, daemonSet, nil)
	assertEqual(t, err, nil)
	for i, expected := range tt.expected {
		assertEqual(t, daemonSet.Spec.Template.Spec.Containers[i].Image, expected)
	}
}

func makeDeployment(name string, podSpec corev1.PodSpec) *appsv1.Deployment {
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

func makeDaemonSet(name string, podSpec corev1.PodSpec) *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			Kind: "DaemonSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: appsv1.DaemonSetSpec{
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

type updateImageSpecTest struct {
	name     string
	in       string
	registry servingv1alpha1.Registry
	expected caching.ImageSpec
}

var updateImageSpecTests = []updateImageSpecTest{
	{
		name: "UsesNameFromDefault",
		in:   "gcr.io/knative-releases/github.com/knative/serving/cmd/queue@sha256:1e40c99ff5977daa2d69873fff604c6d09651af1f9ff15aadf8849b3ee77ab45",
		registry: servingv1alpha1.Registry{
			Default: "new-registry.io/test/path/${NAME}:new-tag",
		},
		expected: caching.ImageSpec{
			Image: "new-registry.io/test/path/UsesNameFromDefault:new-tag",
		},
	},
	{
		name: "AddsImagePullSecrets",
		in:   "gcr.io/knative-releases/github.com/knative/serving/cmd/queue@sha256:1e40c99ff5977daa2d69873fff604c6d09651af1f9ff15aadf8849b3ee77ab45",
		registry: servingv1alpha1.Registry{
			ImagePullSecrets: []corev1.LocalObjectReference{
				{Name: "new-secret"},
			},
		},
		expected: caching.ImageSpec{
			Image: "gcr.io/knative-releases/github.com/knative/serving/cmd/queue@sha256:1e40c99ff5977daa2d69873fff604c6d09651af1f9ff15aadf8849b3ee77ab45",
			ImagePullSecrets: []corev1.LocalObjectReference{
				{Name: "new-secret"},
			},
		},
	},
}

func TestImageTransform(t *testing.T) {
	for _, tt := range updateImageSpecTests {
		t.Run(tt.name, func(t *testing.T) {
			runImageTransformTest(t, &tt)
		})
	}
}
func runImageTransformTest(t *testing.T, tt *updateImageSpecTest) {
	unstructuredImage := makeUnstructured(t, makeImage(t, tt))
	instance := &servingv1alpha1.KnativeServing{
		Spec: servingv1alpha1.KnativeServingSpec{
			Registry: tt.registry,
		},
	}
	imageTransform := ImageTransform(instance, log)
	imageTransform(&unstructuredImage)
	validateUnstructedImageChanged(t, tt, &unstructuredImage)
}

func validateUnstructedImageChanged(t *testing.T, tt *updateImageSpecTest, u *unstructured.Unstructured) {
	var image = &caching.Image{}
	err := scheme.Scheme.Convert(u, image, nil)
	assertEqual(t, err, nil)
	assertDeepEqual(t, image.Spec, tt.expected)
}

func makeImage(t *testing.T, tt *updateImageSpecTest) *caching.Image {
	return &caching.Image{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "caching.internal.knative.dev/v1alpha1",
			Kind:       "Image",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: tt.name,
		},
		Spec: caching.ImageSpec{
			Image: tt.in,
		},
	}
}

type addImagePullSecretsTest struct {
	name            string
	existingSecrets []corev1.LocalObjectReference
	registry        servingv1alpha1.Registry
	expectedSecrets []corev1.LocalObjectReference
}

var addImagePullSecretsTests = []addImagePullSecretsTest{
	{
		name:            "LeavesSecretsEmptyByDefault",
		existingSecrets: nil,
		registry:        servingv1alpha1.Registry{},
		expectedSecrets: nil,
	},
	{
		name:            "AddsImagePullSecrets",
		existingSecrets: nil,
		registry: servingv1alpha1.Registry{
			ImagePullSecrets: []corev1.LocalObjectReference{{Name: "new-secret"}},
		},
		expectedSecrets: []corev1.LocalObjectReference{{Name: "new-secret"}},
	},
	{
		name:            "SupportsMultipleImagePullSecrets",
		existingSecrets: nil,
		registry: servingv1alpha1.Registry{
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
		registry: servingv1alpha1.Registry{
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
	unstructuredDeployment := makeUnstructured(t, makeDeployment(tt.name, corev1.PodSpec{ImagePullSecrets: tt.existingSecrets}))
	instance := &servingv1alpha1.KnativeServing{
		Spec: servingv1alpha1.KnativeServingSpec{
			Registry: tt.registry,
		},
	}
	deploymentTransform := ImageTransform(instance, log)
	deploymentTransform(&unstructuredDeployment)

	var deployment = &appsv1.Deployment{}
	err := scheme.Scheme.Convert(&unstructuredDeployment, deployment, nil)

	assertEqual(t, err, nil)
	assertDeepEqual(t, deployment.Spec.Template.Spec.ImagePullSecrets, tt.expectedSecrets)

	unstructuredDaemonSet := makeUnstructured(t, makeDaemonSet(tt.name, corev1.PodSpec{ImagePullSecrets: tt.existingSecrets}))
	daemonSetinstance := &servingv1alpha1.KnativeServing{
		Spec: servingv1alpha1.KnativeServingSpec{
			Registry: tt.registry,
		},
	}
	daemonSetTransform := ImageTransform(daemonSetinstance, log)
	daemonSetTransform(&unstructuredDaemonSet)

	var daemonSet = &appsv1.DaemonSet{}
	err = scheme.Scheme.Convert(&unstructuredDaemonSet, daemonSet, nil)

	assertEqual(t, err, nil)
	assertDeepEqual(t, daemonSet.Spec.Template.Spec.ImagePullSecrets, tt.expectedSecrets)
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
