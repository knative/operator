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

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	caching "knative.dev/caching/pkg/apis/caching/v1alpha1"
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
	util "knative.dev/operator/pkg/reconciler/common/testing"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
)

var log = zap.NewNop().Sugar()

type updateImageTest struct {
	name       string
	containers []corev1.Container
	registry   v1alpha1.Registry
	expected   []corev1.Container
}

var updateImageTests = []updateImageTest{
	{
		name: "UsesNameFromDefault",
		containers: []corev1.Container{{
			Name:  "queue",
			Image: "gcr.io/knative-releases/github.com/knative/serving/cmd/queue@sha256:1e40c99ff5977daa2d69873fff604c6d09651af1f9ff15aadf8849b3ee77ab45"},
		},
		registry: v1alpha1.Registry{
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
		registry: v1alpha1.Registry{
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
			Image: "gcr.io/knative-releases/github.com/knative/serving/cmd/queue@sha256:1e40c99ff5977daa2d69873fff604c6d09651af1f9ff15aadf8849b3ee77ab45"},
		},
		registry: v1alpha1.Registry{
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
		registry: v1alpha1.Registry{
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
		registry: v1alpha1.Registry{},
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
		registry: v1alpha1.Registry{
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
		registry: v1alpha1.Registry{
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
		registry: v1alpha1.Registry{
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
		registry: v1alpha1.Registry{
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
		registry: v1alpha1.Registry{
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
		registry: v1alpha1.Registry{
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

func TestResourceTransform(t *testing.T) {
	for _, tt := range updateImageTests {
		t.Run(tt.name, func(t *testing.T) {
			runResourceTransformTest(t, &tt)
		})
	}
}

func runResourceTransformTest(t *testing.T, tt *updateImageTest) {
	// test for deployment
	unstructuredDeployment := util.MakeUnstructured(t, util.MakeDeployment(tt.name, corev1.PodSpec{Containers: tt.containers}))
	deploymentTransform := ImageTransform(&tt.registry, log)
	deploymentTransform(&unstructuredDeployment)
	validateUnstructuredDeploymentChanged(t, tt, &unstructuredDeployment)

	// test for daemonSet
	unstructuredDaemonSet := util.MakeUnstructured(t, makeDaemonSet(tt.name, corev1.PodSpec{Containers: tt.containers}))
	daemonSetTransform := ImageTransform(&tt.registry, log)
	daemonSetTransform(&unstructuredDaemonSet)
	validateUnstructuredDaemonSetChanged(t, tt, &unstructuredDaemonSet)

	// test for job
	unstructuredJob := util.MakeUnstructured(t, makeJob(tt.name, corev1.PodSpec{Containers: tt.containers}))
	jobTransform := ImageTransform(&tt.registry, log)
	jobTransform(&unstructuredJob)
	validateUnstructuredJobChanged(t, tt, &unstructuredJob)
}

func validateUnstructuredDeploymentChanged(t *testing.T, tt *updateImageTest, u *unstructured.Unstructured) {
	var deployment = &appsv1.Deployment{}
	err := scheme.Scheme.Convert(u, deployment, nil)
	util.AssertEqual(t, err, nil)
	util.AssertDeepEqual(t, deployment.Spec.Template.Spec.Containers, tt.expected)
}

func validateUnstructuredDaemonSetChanged(t *testing.T, tt *updateImageTest, u *unstructured.Unstructured) {
	var daemonSet = &appsv1.DaemonSet{}
	err := scheme.Scheme.Convert(u, daemonSet, nil)
	util.AssertEqual(t, err, nil)
	util.AssertDeepEqual(t, daemonSet.Spec.Template.Spec.Containers, tt.expected)
}

func validateUnstructuredJobChanged(t *testing.T, tt *updateImageTest, u *unstructured.Unstructured) {
	var job = &batchv1.Job{}
	err := scheme.Scheme.Convert(u, job, nil)
	util.AssertEqual(t, err, nil)
	util.AssertDeepEqual(t, job.Spec.Template.Spec.Containers, tt.expected)
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

func makeJob(name string, podSpec corev1.PodSpec) *batchv1.Job {
	return &batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			Kind: "DaemonSet",
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

type updateImageSpecTest struct {
	name     string
	in       string
	registry v1alpha1.Registry
	expected caching.ImageSpec
}

var updateImageSpecTests = []updateImageSpecTest{
	{
		name: "UsesNameFromDefault",
		in:   "gcr.io/knative-releases/github.com/knative/serving/cmd/queue@sha256:1e40c99ff5977daa2d69873fff604c6d09651af1f9ff15aadf8849b3ee77ab45",
		registry: v1alpha1.Registry{
			Default: "new-registry.io/test/path/${NAME}:new-tag",
		},
		expected: caching.ImageSpec{
			Image: "new-registry.io/test/path/UsesNameFromDefault:new-tag",
		},
	},
	{
		name: "AddsImagePullSecrets",
		in:   "gcr.io/knative-releases/github.com/knative/serving/cmd/queue@sha256:1e40c99ff5977daa2d69873fff604c6d09651af1f9ff15aadf8849b3ee77ab45",
		registry: v1alpha1.Registry{
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
	unstructuredImage := util.MakeUnstructured(t, makeImage(t, tt))
	instance := &v1alpha1.KnativeServing{
		Spec: v1alpha1.KnativeServingSpec{
			CommonSpec: v1alpha1.CommonSpec{
				Registry: tt.registry,
			},
		},
	}
	imageTransform := ImageTransform(&instance.Spec.Registry, log)
	imageTransform(&unstructuredImage)
	validateUnstructuredImageChanged(t, tt, &unstructuredImage)
}

func validateUnstructuredImageChanged(t *testing.T, tt *updateImageSpecTest, u *unstructured.Unstructured) {
	var image = &caching.Image{}
	err := scheme.Scheme.Convert(u, image, nil)
	util.AssertEqual(t, err, nil)
	util.AssertDeepEqual(t, image.Spec, tt.expected)
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
	registry        v1alpha1.Registry
	expectedSecrets []corev1.LocalObjectReference
}

var addImagePullSecretsTests = []addImagePullSecretsTest{
	{
		name:            "LeavesSecretsEmptyByDefault",
		existingSecrets: nil,
		registry:        v1alpha1.Registry{},
		expectedSecrets: nil,
	},
	{
		name:            "AddsImagePullSecrets",
		existingSecrets: nil,
		registry: v1alpha1.Registry{
			ImagePullSecrets: []corev1.LocalObjectReference{{Name: "new-secret"}},
		},
		expectedSecrets: []corev1.LocalObjectReference{{Name: "new-secret"}},
	},
	{
		name:            "SupportsMultipleImagePullSecrets",
		existingSecrets: nil,
		registry: v1alpha1.Registry{
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
		registry: v1alpha1.Registry{
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
	unstructuredDeployment := util.MakeUnstructured(t, util.MakeDeployment(tt.name, corev1.PodSpec{ImagePullSecrets: tt.existingSecrets}))
	deploymentTransform := ImageTransform(&tt.registry, log)
	deploymentTransform(&unstructuredDeployment)

	var deployment = &appsv1.Deployment{}
	err := scheme.Scheme.Convert(&unstructuredDeployment, deployment, nil)

	util.AssertEqual(t, err, nil)
	util.AssertDeepEqual(t, deployment.Spec.Template.Spec.ImagePullSecrets, tt.expectedSecrets)

	unstructuredDaemonSet := util.MakeUnstructured(t, makeDaemonSet(tt.name, corev1.PodSpec{ImagePullSecrets: tt.existingSecrets}))
	daemonSetTransform := ImageTransform(&tt.registry, log)
	daemonSetTransform(&unstructuredDaemonSet)

	var daemonSet = &appsv1.DaemonSet{}
	err = scheme.Scheme.Convert(&unstructuredDaemonSet, daemonSet, nil)

	util.AssertEqual(t, err, nil)
	util.AssertDeepEqual(t, daemonSet.Spec.Template.Spec.ImagePullSecrets, tt.expectedSecrets)
}
