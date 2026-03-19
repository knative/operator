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
	"os"

	mf "github.com/manifestival/manifestival"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
	"knative.dev/pkg/version"
)

// KubernetesMinVersionTransform injects KUBERNETES_MIN_VERSION into all workloads
// managed by this operator instance so operand components can honor the override.
func KubernetesMinVersionTransform() mf.Transformer {
	minVersion := os.Getenv(version.KubernetesMinVersionKey)
	if minVersion == "" {
		return func(_ *unstructured.Unstructured) error {
			return nil
		}
	}

	minVersionEnv := []corev1.EnvVar{{
		Name:  version.KubernetesMinVersionKey,
		Value: minVersion,
	}}

	return func(u *unstructured.Unstructured) error {
		var podSpec *corev1.PodSpec

		switch u.GetKind() {
		case "Deployment":
			deployment := &appsv1.Deployment{}
			if err := scheme.Scheme.Convert(u, deployment, nil); err != nil {
				return err
			}
			podSpec = &deployment.Spec.Template.Spec
			applyMinVersionEnvVar(podSpec, minVersionEnv)
			if err := scheme.Scheme.Convert(deployment, u, nil); err != nil {
				return err
			}
		case "StatefulSet":
			ss := &appsv1.StatefulSet{}
			if err := scheme.Scheme.Convert(u, ss, nil); err != nil {
				return err
			}
			podSpec = &ss.Spec.Template.Spec
			applyMinVersionEnvVar(podSpec, minVersionEnv)
			if err := scheme.Scheme.Convert(ss, u, nil); err != nil {
				return err
			}
		case "DaemonSet":
			ds := &appsv1.DaemonSet{}
			if err := scheme.Scheme.Convert(u, ds, nil); err != nil {
				return err
			}
			podSpec = &ds.Spec.Template.Spec
			applyMinVersionEnvVar(podSpec, minVersionEnv)
			if err := scheme.Scheme.Convert(ds, u, nil); err != nil {
				return err
			}
		case "Job":
			job := &batchv1.Job{}
			if err := scheme.Scheme.Convert(u, job, nil); err != nil {
				return err
			}
			podSpec = &job.Spec.Template.Spec
			applyMinVersionEnvVar(podSpec, minVersionEnv)
			if err := scheme.Scheme.Convert(job, u, nil); err != nil {
				return err
			}
		default:
			return nil
		}

		// Avoid superfluous updates from converted zero defaults.
		u.SetCreationTimestamp(metav1.Time{})
		return nil
	}
}

func applyMinVersionEnvVar(podSpec *corev1.PodSpec, minVersionEnv []corev1.EnvVar) {
	for i := range podSpec.Containers {
		mergeEnv(&minVersionEnv, &podSpec.Containers[i].Env)
	}
	for i := range podSpec.InitContainers {
		mergeEnv(&minVersionEnv, &podSpec.InitContainers[i].Env)
	}
}
