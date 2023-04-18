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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	caching "knative.dev/caching/pkg/apis/caching/v1alpha1"
	"knative.dev/operator/pkg/apis/operator/base"
	"knative.dev/operator/pkg/apis/operator/v1beta1"
	util "knative.dev/operator/pkg/reconciler/common/testing"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
)

var log = zap.NewNop().Sugar()

func TestResourceTransform(t *testing.T) {
	for _, tt := range []struct {
		name       string
		containers []corev1.Container
		registry   base.Registry
		expected   []corev1.Container
	}{{
		name: "UsesNameFromDefault",
		containers: []corev1.Container{{
			Name:  "queue",
			Image: "gcr.io/knative-releases/github.com/knative/serving/cmd/queue@sha256:1e40c99ff5977daa2d69873fff604c6d09651af1f9ff15aadf8849b3ee77ab45",
		}},
		registry: base.Registry{
			Default: "new-registry.io/test/path/${NAME}:new-tag",
		},
		expected: []corev1.Container{{
			Name:  "queue",
			Image: "new-registry.io/test/path/queue:new-tag",
		}},
	}, {
		name: "UsesContainerNamePerContainer",
		containers: []corev1.Container{{
			Name:  "container1",
			Image: "gcr.io/cmd/queue:test",
		}, {
			Name:  "container2",
			Image: "gcr.io/cmd/queue:test",
		}},
		registry: base.Registry{
			Override: map[string]string{
				"container1": "new-registry.io/test/path/new-container-1:new-tag",
				"container2": "new-registry.io/test/path/new-container-2:new-tag",
			},
		},
		expected: []corev1.Container{{
			Name:  "container1",
			Image: "new-registry.io/test/path/new-container-1:new-tag",
		}, {
			Name:  "container2",
			Image: "new-registry.io/test/path/new-container-2:new-tag",
		}},
	}, {
		name: "UsesOverrideFromDefault",
		containers: []corev1.Container{{
			Name:  "queue",
			Image: "gcr.io/knative-releases/github.com/knative/serving/cmd/queue@sha256:1e40c99ff5977daa2d69873fff604c6d09651af1f9ff15aadf8849b3ee77ab45",
		}},
		registry: base.Registry{
			Default: "new-registry.io/test/path/${NAME}:new-tag",
			Override: map[string]string{
				"queue": "new-registry.io/test/path/new-value:new-override-tag",
			},
		},
		expected: []corev1.Container{{
			Name:  "queue",
			Image: "new-registry.io/test/path/new-value:new-override-tag",
		}},
	}, {
		name: "NoChangeOverrideWithDifferentName",
		containers: []corev1.Container{{
			Name:  "image",
			Image: "docker.io/name/image:tag2",
		}},
		registry: base.Registry{
			Override: map[string]string{
				"Unused": "new-registry.io/test/path",
			},
		},
		expected: []corev1.Container{{
			Name:  "image",
			Image: "docker.io/name/image:tag2",
		}},
	}, {
		name: "NoChange",
		containers: []corev1.Container{{
			Name:  "queue",
			Image: "gcr.io/knative-releases/github.com/knative/eventing/cmd/queue@sha256:1e40c99ff5977daa2d69873fff604c6d09651af1f9ff15aadf8849b3ee77ab45",
		}},
		registry: base.Registry{},
		expected: []corev1.Container{{
			Name:  "queue",
			Image: "gcr.io/knative-releases/github.com/knative/eventing/cmd/queue@sha256:1e40c99ff5977daa2d69873fff604c6d09651af1f9ff15aadf8849b3ee77ab45",
		}},
	}, {
		name: "OverrideEnvVarImage",
		containers: []corev1.Container{{
			Env: []corev1.EnvVar{{Name: "SOME_IMAGE", Value: "gcr.io/foo/bar"}},
		}},
		registry: base.Registry{
			Override: map[string]string{
				"SOME_IMAGE": "docker.io/my/overridden-image",
			},
		},
		expected: []corev1.Container{{
			Env: []corev1.EnvVar{{Name: "SOME_IMAGE", Value: "docker.io/my/overridden-image"}},
		}},
	}, {
		name: "NoOverrideEnvVarImage",
		containers: []corev1.Container{{
			Env: []corev1.EnvVar{{Name: "SOME_IMAGE", Value: "gcr.io/foo/bar"}},
		}},
		registry: base.Registry{
			Override: map[string]string{
				"OTHER_IMAGE": "docker.io/my/overridden-image",
			},
		},
		expected: []corev1.Container{{
			Env: []corev1.EnvVar{{Name: "SOME_IMAGE", Value: "gcr.io/foo/bar"}},
		}},
	}, {
		name: "NoOverrideEnvVarImageAndContainerImageBoth",
		containers: []corev1.Container{{
			Name:  "queue",
			Image: "gcr.io/knative-releases/github.com/knative/eventing/cmd/queue@sha256:1e40c99ff5977daa2d69873fff604c6d09651af1f9ff15aadf8849b3ee77ab45",
			Env:   []corev1.EnvVar{{Name: "SOME_IMAGE", Value: "gcr.io/foo/bar"}},
		}},
		registry: base.Registry{
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
	}, {
		name: "OverrideWithDeploymentContainer",
		containers: []corev1.Container{{
			Name:  "container1",
			Image: "gcr.io/cmd/queue:test",
		}, {
			Name:  "container2",
			Image: "gcr.io/cmd/queue:test",
		}},
		registry: base.Registry{
			Override: map[string]string{
				"container1": "new-registry.io/test/path/new-container-1:new-tag",
				"container2": "new-registry.io/test/path/new-container-2:new-tag",
				"OverrideWithDeploymentContainer/container1": "new-registry.io/test/path/OverrideWithDeploymentContainer/container-1:new-tag",
				"OverrideWithDeploymentContainer/container2": "new-registry.io/test/path/OverrideWithDeploymentContainer/container-2:new-tag",
			},
		},
		expected: []corev1.Container{{
			Name:  "container1",
			Image: "new-registry.io/test/path/OverrideWithDeploymentContainer/container-1:new-tag",
		}, {
			Name:  "container2",
			Image: "new-registry.io/test/path/OverrideWithDeploymentContainer/container-2:new-tag",
		}},
	}, {
		name: "OverridePartialWithDeploymentContainer",
		containers: []corev1.Container{{
			Name:  "container1",
			Image: "gcr.io/cmd/queue:test",
		}, {
			Name:  "container2",
			Image: "gcr.io/cmd/queue:test",
		}},
		registry: base.Registry{
			Override: map[string]string{
				"container1": "new-registry.io/test/path/new-container-1:new-tag",
				"container2": "new-registry.io/test/path/new-container-2:new-tag",
				"OverridePartialWithDeploymentContainer/container1": "new-registry.io/test/path/OverridePartialWithDeploymentContainer/container-1:new-tag",
			},
		},
		expected: []corev1.Container{{
			Name:  "container1",
			Image: "new-registry.io/test/path/OverridePartialWithDeploymentContainer/container-1:new-tag",
		}, {
			Name:  "container2",
			Image: "new-registry.io/test/path/new-container-2:new-tag",
		}},
	}, {
		name: "OverrideWithDeploymentName",
		containers: []corev1.Container{{
			Name:  "container1",
			Image: "gcr.io/cmd/queue:test",
		}, {
			Name:  "container2",
			Image: "gcr.io/cmd/queue:test",
		}},
		registry: base.Registry{
			Override: map[string]string{
				"OverrideWithDeploymentName/container1": "new-registry.io/test/path/OverrideWithDeploymentName/container-1:new-tag",
				"OverrideWithDeploymentName/container2": "new-registry.io/test/path/OverrideWithDeploymentName/container-2:new-tag",
			},
		},
		expected: []corev1.Container{{
			Name:  "container1",
			Image: "new-registry.io/test/path/OverrideWithDeploymentName/container-1:new-tag",
		}, {
			Name:  "container2",
			Image: "new-registry.io/test/path/OverrideWithDeploymentName/container-2:new-tag",
		}},
	}} {
		t.Run(tt.name, func(t *testing.T) {
			transform := ImageTransform(&tt.registry, log)
			podSpec := corev1.PodSpec{Containers: tt.containers}

			// test for deployment
			unstructuredDeployment := util.MakeUnstructured(t, util.MakeDeployment(tt.name, podSpec))
			transform(&unstructuredDeployment)
			deployment := &appsv1.Deployment{}
			err := scheme.Scheme.Convert(&unstructuredDeployment, deployment, nil)
			util.AssertEqual(t, err, nil)
			util.AssertDeepEqual(t, deployment.Spec.Template.Spec.Containers, tt.expected)

			// test for daemonSet
			unstructuredDaemonSet := util.MakeUnstructured(t, makeDaemonSet(tt.name, podSpec))
			transform(&unstructuredDaemonSet)
			daemonSet := &appsv1.DaemonSet{}
			err = scheme.Scheme.Convert(&unstructuredDaemonSet, daemonSet, nil)
			util.AssertEqual(t, err, nil)
			util.AssertDeepEqual(t, daemonSet.Spec.Template.Spec.Containers, tt.expected)

			// test for job
			unstructuredJob := util.MakeUnstructured(t, makeJob(tt.name, podSpec))
			transform(&unstructuredJob)
			job := &batchv1.Job{}
			err = scheme.Scheme.Convert(&unstructuredJob, job, nil)
			util.AssertEqual(t, err, nil)
			util.AssertDeepEqual(t, job.Spec.Template.Spec.Containers, tt.expected)

			// test for statefulset
			unstructuredStatefulSet := util.MakeUnstructured(t, makeStatefulSet(tt.name, podSpec))
			transform(&unstructuredStatefulSet)
			statefulSet := &appsv1.StatefulSet{}
			err = scheme.Scheme.Convert(&unstructuredStatefulSet, statefulSet, nil)
			util.AssertEqual(t, err, nil)
			util.AssertDeepEqual(t, job.Spec.Template.Spec.Containers, tt.expected)
		})
	}
}

func TestImageTransform(t *testing.T) {
	for _, tt := range []struct {
		name     string
		in       string
		registry base.Registry
		expected caching.ImageSpec
	}{{
		name: "OverrideImage",
		in:   "gcr.io/knative-releases/github.com/knative/serving/cmd/queue@sha256:1e40c99ff5977daa2d69873fff604c6d09651af1f9ff15aadf8849b3ee77ab45",
		registry: base.Registry{
			Override: map[string]string{
				"OverrideImage": "new-registry.io/test/path/OverrideImage:new-tag",
			},
		},
		expected: caching.ImageSpec{
			Image: "new-registry.io/test/path/OverrideImage:new-tag",
		},
	}, {
		name: "UsesDefaultImageNameWithSha",
		in:   "gcr.io/knative-releases/github.com/knative/serving/cmd/queue@sha256:1e40c99ff5977daa2d69873fff604c6d09651af1f9ff15aadf8849b3ee77ab45",
		registry: base.Registry{
			Default: "new-registry.io/test/path/${NAME}:new-tag",
		},
		expected: caching.ImageSpec{
			Image: "new-registry.io/test/path/queue:new-tag",
		},
	}, {
		name: "UsesDefaultContainerName",
		in:   "badLink",
		registry: base.Registry{
			Default: "new-registry.io/test/path/${NAME}:new-tag",
		},
		expected: caching.ImageSpec{
			Image: "new-registry.io/test/path/UsesDefaultContainerName:new-tag",
		},
	}, {
		name: "UsesDefaultImageNameWithTag",
		in:   "gcr.io/knative-releases/github.com/knative/serving/cmd/queue:v1.2.0",
		registry: base.Registry{
			Default: "new-registry.io/test/path/${NAME}:new-tag",
		},
		expected: caching.ImageSpec{
			Image: "new-registry.io/test/path/queue:new-tag",
		},
	}, {
		name: "AddsImagePullSecrets",
		in:   "gcr.io/knative-releases/github.com/knative/serving/cmd/queue@sha256:1e40c99ff5977daa2d69873fff604c6d09651af1f9ff15aadf8849b3ee77ab45",
		registry: base.Registry{
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
	}} {
		t.Run(tt.name, func(t *testing.T) {
			unstructuredImage := util.MakeUnstructured(t, makeImage(tt.name, tt.in))
			instance := &v1beta1.KnativeServing{
				Spec: v1beta1.KnativeServingSpec{
					CommonSpec: base.CommonSpec{
						Registry: tt.registry,
					},
				},
			}
			ImageTransform(&instance.Spec.Registry, log)(&unstructuredImage)
			image := &caching.Image{}
			err := scheme.Scheme.Convert(&unstructuredImage, image, nil)
			util.AssertEqual(t, err, nil)
			util.AssertDeepEqual(t, image.Spec, tt.expected)
		})
	}
}

func TestImagePullSecrets(t *testing.T) {
	for _, tt := range []struct {
		name            string
		existingSecrets []corev1.LocalObjectReference
		registry        base.Registry
		expectedSecrets []corev1.LocalObjectReference
	}{{
		name:            "LeavesSecretsEmptyByDefault",
		existingSecrets: nil,
		registry:        base.Registry{},
		expectedSecrets: nil,
	}, {
		name:            "AddsImagePullSecrets",
		existingSecrets: nil,
		registry: base.Registry{
			ImagePullSecrets: []corev1.LocalObjectReference{{Name: "new-secret"}},
		},
		expectedSecrets: []corev1.LocalObjectReference{{Name: "new-secret"}},
	}, {
		name:            "SupportsMultipleImagePullSecrets",
		existingSecrets: nil,
		registry: base.Registry{
			ImagePullSecrets: []corev1.LocalObjectReference{
				{Name: "new-secret-1"},
				{Name: "new-secret-2"},
			},
		},
		expectedSecrets: []corev1.LocalObjectReference{
			{Name: "new-secret-1"},
			{Name: "new-secret-2"},
		},
	}, {
		name:            "MergesAdditionalSecretsWithAnyPreexisting",
		existingSecrets: []corev1.LocalObjectReference{{Name: "existing-secret"}},
		registry: base.Registry{
			ImagePullSecrets: []corev1.LocalObjectReference{
				{Name: "new-secret"},
			},
		},
		expectedSecrets: []corev1.LocalObjectReference{
			{Name: "existing-secret"},
			{Name: "new-secret"},
		},
	}} {
		t.Run(tt.name, func(t *testing.T) {
			transform := ImageTransform(&tt.registry, log)
			podSpec := corev1.PodSpec{ImagePullSecrets: tt.existingSecrets}

			unstructuredDeployment := util.MakeUnstructured(t, util.MakeDeployment(tt.name, podSpec))
			transform(&unstructuredDeployment)
			deployment := &appsv1.Deployment{}
			err := scheme.Scheme.Convert(&unstructuredDeployment, deployment, nil)
			util.AssertEqual(t, err, nil)
			util.AssertDeepEqual(t, deployment.Spec.Template.Spec.ImagePullSecrets, tt.expectedSecrets)

			unstructuredDaemonSet := util.MakeUnstructured(t, makeDaemonSet(tt.name, podSpec))
			daemonSetTransform := ImageTransform(&tt.registry, log)
			daemonSetTransform(&unstructuredDaemonSet)
			daemonSet := &appsv1.DaemonSet{}
			err = scheme.Scheme.Convert(&unstructuredDaemonSet, daemonSet, nil)
			util.AssertEqual(t, err, nil)
			util.AssertDeepEqual(t, daemonSet.Spec.Template.Spec.ImagePullSecrets, tt.expectedSecrets)

			unstructuredStatefulSet := util.MakeUnstructured(t, makeStatefulSet(tt.name, podSpec))
			statefulSetTransform := ImageTransform(&tt.registry, log)
			statefulSetTransform(&unstructuredStatefulSet)
			statefulSet := &appsv1.StatefulSet{}
			err = scheme.Scheme.Convert(&unstructuredStatefulSet, statefulSet, nil)
			util.AssertEqual(t, err, nil)
			util.AssertDeepEqual(t, statefulSet.Spec.Template.Spec.ImagePullSecrets, tt.expectedSecrets)
		})
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

func makeStatefulSet(name string, podSpec corev1.PodSpec) *appsv1.StatefulSet {
	return &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			Kind: "StatefulSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: appsv1.StatefulSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: podSpec,
			},
		},
	}
}

func makeJob(name string, podSpec corev1.PodSpec) *batchv1.Job {
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

func makeImage(name, image string) *caching.Image {
	return &caching.Image{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "caching.internal.knative.dev/v1alpha1",
			Kind:       "Image",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: caching.ImageSpec{
			Image: image,
		},
	}
}
