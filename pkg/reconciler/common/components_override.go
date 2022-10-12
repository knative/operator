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
	mf "github.com/manifestival/manifestival"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"

	"knative.dev/operator/pkg/apis/operator/base"
)

// DeploymentTransform transforms deployments based on the configuration in `spec.deployment`.
func ComponentsTransform(obj base.KComponent, log *zap.SugaredLogger) mf.Transformer {
	overrides := obj.GetSpec().GetComponentsOverride()
	if overrides == nil {
		return nil
	}
	return func(u *unstructured.Unstructured) error {
		for _, override := range overrides {
			var obj metav1.Object
			var ps *corev1.PodTemplateSpec

			if u.GetKind() == "Deployment" && u.GetName() == override.Name {
				deployment := &appsv1.Deployment{}
				if err := scheme.Scheme.Convert(u, deployment, nil); err != nil {
					return err
				}
				obj = deployment
				ps = &deployment.Spec.Template
				replaceReplicas(&override, deployment.Spec.Replicas)
			}
			if u.GetKind() == "StatefulSet" && u.GetName() == override.Name {
				ss := &appsv1.StatefulSet{}
				if err := scheme.Scheme.Convert(u, ss, nil); err != nil {
					return err
				}
				obj = ss
				ps = &ss.Spec.Template
				replaceReplicas(&override, ss.Spec.Replicas)
			}

			replaceLabels(&override, obj, ps)
			replaceAnnotations(&override, obj, ps)
			replaceNodeSelector(&override, ps)
			replaceTolerations(&override, ps)
			replaceAffinities(&override, ps)
			replaceResources(&override, ps)
			replaceEnv(&override, ps)

			if err := scheme.Scheme.Convert(obj, u, nil); err != nil {
				return err
			}

			// Avoid superfluous updates from converted zero defaults
			u.SetCreationTimestamp(metav1.Time{})
		}
		return nil
	}
}

func replaceAnnotations(override *base.ComponentOverride, obj metav1.Object, ps *corev1.PodTemplateSpec) {
	if obj.GetAnnotations() == nil {
		obj.SetAnnotations(map[string]string{})
	}
	if ps.GetAnnotations() == nil {
		ps.SetAnnotations(map[string]string{})
	}
	for key, val := range override.Annotations {
		obj.GetAnnotations()[key] = val
		ps.Annotations[key] = val
	}
}

func replaceLabels(override *base.ComponentOverride, obj metav1.Object, ps *corev1.PodTemplateSpec) {
	if obj.GetLabels() == nil {
		obj.SetLabels(map[string]string{})
	}
	if ps.GetLabels() == nil {
		ps.Labels = map[string]string{}
	}
	for key, val := range override.Labels {
		obj.GetLabels()[key] = val
		ps.Labels[key] = val
	}
}

func replaceReplicas(override *base.ComponentOverride, replicas *int32) {
	if override.Replicas != nil {
		*replicas = *override.Replicas
	}
}

func replaceNodeSelector(override *base.ComponentOverride, ps *corev1.PodTemplateSpec) {
	if len(override.NodeSelector) > 0 {
		ps.Spec.NodeSelector = override.NodeSelector
	}
}

func replaceTolerations(override *base.ComponentOverride, ps *corev1.PodTemplateSpec) {
	if len(override.Tolerations) > 0 {
		ps.Spec.Tolerations = override.Tolerations
	}
}

func replaceAffinities(override *base.ComponentOverride, ps *corev1.PodTemplateSpec) {
	if override.Affinity != nil {
		ps.Spec.Affinity = override.Affinity
	}
}

func replaceResources(override *base.ComponentOverride, ps *corev1.PodTemplateSpec) {
	if len(override.Resources) > 0 {
		containers := ps.Spec.Containers
		for i := range containers {
			if override := find(override.Resources, containers[i].Name); override != nil {
				merge(&override.Limits, &containers[i].Resources.Limits)
				merge(&override.Requests, &containers[i].Resources.Requests)
			}
		}
	}
}

func replaceEnv(override *base.ComponentOverride, ps *corev1.PodTemplateSpec) {
	if len(override.Env) > 0 {
		containers := ps.Spec.Containers
		for i := range containers {
			if override := findEnvOverride(override.Env, containers[i].Name); override != nil {
				mergeEnv(&override.EnvVars, &containers[i].Env)
			}
		}
	}
}
