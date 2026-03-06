/*
Copyright 2026 The Knative Authors

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
	"fmt"
	"testing"

	mf "github.com/manifestival/manifestival"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
	util "knative.dev/operator/pkg/reconciler/common/testing"
	"knative.dev/pkg/version"
)

func TestKubernetesMinVersionTransformInjectsEnvVar(t *testing.T) {
	t.Setenv(version.KubernetesMinVersionKey, "v1.25.0")

	makePodSpec := func() corev1.PodSpec {
		return corev1.PodSpec{
			Containers: []corev1.Container{{
				Name: "controller",
				Env: []corev1.EnvVar{
					{Name: "EXISTING", Value: "1"},
					{Name: version.KubernetesMinVersionKey, Value: "v1.20.0"},
				},
			}, {
				Name: "webhook",
			}},
			InitContainers: []corev1.Container{{
				Name: "init-a",
			}, {
				Name: "init-b",
				Env: []corev1.EnvVar{{Name: version.KubernetesMinVersionKey, Value: "v1.19.0"}},
			}},
		}
	}

	testCases := []struct {
		name string
		obj  interface{}
	}{
		{
			name: "Deployment",
			obj:  util.MakeDeployment("controller", makePodSpec()),
		},
		{
			name: "StatefulSet",
			obj:  util.MakeStatefulSet("controller", makePodSpec()),
		},
		{
			name: "DaemonSet",
			obj:  util.MakeDaemonSet("controller", makePodSpec()),
		},
		{
			name: "Job",
			obj:  util.MakeJob("controller", makePodSpec()),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			u := util.MakeUnstructured(t, tc.obj)
			manifest, err := mf.ManifestFrom(mf.Slice([]unstructured.Unstructured{u}))
			if err != nil {
				t.Fatalf("Failed to create manifest: %v", err)
			}

			manifest, err = manifest.Transform(KubernetesMinVersionTransform())
			if err != nil {
				t.Fatalf("Failed to transform manifest: %v", err)
			}

			podSpec, err := podSpecFromResource(manifest.Resources()[0])
			if err != nil {
				t.Fatalf("Failed to extract pod spec: %v", err)
			}

			for _, c := range podSpec.Containers {
				if !hasEnv(c.Env, version.KubernetesMinVersionKey, "v1.25.0") {
					t.Fatalf("container %q misses %s=v1.25.0", c.Name, version.KubernetesMinVersionKey)
				}
			}
			for _, c := range podSpec.InitContainers {
				if !hasEnv(c.Env, version.KubernetesMinVersionKey, "v1.25.0") {
					t.Fatalf("init container %q misses %s=v1.25.0", c.Name, version.KubernetesMinVersionKey)
				}
			}
			if !hasEnv(podSpec.Containers[0].Env, "EXISTING", "1") {
				t.Fatalf("existing env var was removed")
			}
		})
	}
}

func TestKubernetesMinVersionTransformNoopWhenEnvUnset(t *testing.T) {
	t.Setenv(version.KubernetesMinVersionKey, "")

	deployment := util.MakeDeployment("controller", corev1.PodSpec{
		Containers: []corev1.Container{{
			Name: "controller",
			Env:  []corev1.EnvVar{{Name: "EXISTING", Value: "1"}},
		}},
		InitContainers: []corev1.Container{{
			Name: "init-a",
			Env:  []corev1.EnvVar{{Name: "EXISTING_INIT", Value: "1"}},
		}},
	})
	u := util.MakeUnstructured(t, deployment)
	manifest, err := mf.ManifestFrom(mf.Slice([]unstructured.Unstructured{u}))
	if err != nil {
		t.Fatalf("Failed to create manifest: %v", err)
	}

	manifest, err = manifest.Transform(KubernetesMinVersionTransform())
	if err != nil {
		t.Fatalf("Failed to transform manifest: %v", err)
	}

	var got appsv1.Deployment
	if err := scheme.Scheme.Convert(&manifest.Resources()[0], &got, nil); err != nil {
		t.Fatalf("Failed to convert deployment: %v", err)
	}

	if hasEnv(got.Spec.Template.Spec.Containers[0].Env, version.KubernetesMinVersionKey, "") {
		t.Fatalf("unexpected %s env var was injected", version.KubernetesMinVersionKey)
	}
	if !hasEnv(got.Spec.Template.Spec.Containers[0].Env, "EXISTING", "1") {
		t.Fatalf("existing env var was changed")
	}
	if hasEnv(got.Spec.Template.Spec.InitContainers[0].Env, version.KubernetesMinVersionKey, "") {
		t.Fatalf("unexpected %s env var was injected into init container", version.KubernetesMinVersionKey)
	}
	if !hasEnv(got.Spec.Template.Spec.InitContainers[0].Env, "EXISTING_INIT", "1") {
		t.Fatalf("existing init container env var was changed")
	}
}

func hasEnv(envs []corev1.EnvVar, name, value string) bool {
	for _, env := range envs {
		if env.Name == name {
			return env.Value == value
		}
	}
	return false
}

func podSpecFromResource(u unstructured.Unstructured) (corev1.PodSpec, error) {
	switch u.GetKind() {
	case "Deployment":
		var d appsv1.Deployment
		if err := scheme.Scheme.Convert(&u, &d, nil); err != nil {
			return corev1.PodSpec{}, err
		}
		return d.Spec.Template.Spec, nil
	case "StatefulSet":
		var s appsv1.StatefulSet
		if err := scheme.Scheme.Convert(&u, &s, nil); err != nil {
			return corev1.PodSpec{}, err
		}
		return s.Spec.Template.Spec, nil
	case "DaemonSet":
		var d appsv1.DaemonSet
		if err := scheme.Scheme.Convert(&u, &d, nil); err != nil {
			return corev1.PodSpec{}, err
		}
		return d.Spec.Template.Spec, nil
	case "Job":
		var j batchv1.Job
		if err := scheme.Scheme.Convert(&u, &j, nil); err != nil {
			return corev1.PodSpec{}, err
		}
		return j.Spec.Template.Spec, nil
	default:
		return corev1.PodSpec{}, fmt.Errorf("unsupported kind: %s", u.GetKind())
	}
}
