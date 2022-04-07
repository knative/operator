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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
	"knative.dev/operator/pkg/apis/operator/base"
)

// DeploymentTransform transforms deployments based on the configuration in `spec.deployment`.
func DeploymentsTransform(obj base.KComponent, log *zap.SugaredLogger) mf.Transformer {
	overrides := obj.GetSpec().GetDeploymentOverride()
	if overrides == nil {
		return nil
	}
	return func(u *unstructured.Unstructured) error {
		for _, override := range overrides {
			if u.GetKind() == "Deployment" && u.GetName() == override.Name {

				deployment := &appsv1.Deployment{}
				if err := scheme.Scheme.Convert(u, deployment, nil); err != nil {
					return err
				}
				replaceLabels(&override, deployment)
				replaceAnnotations(&override, deployment)
				replaceReplicas(&override, deployment)
				replaceNodeSelector(&override, deployment)
				replaceTolerations(&override, deployment)
				replaceAffinities(&override, deployment)
				replaceResources(&override, deployment)
				if err := scheme.Scheme.Convert(deployment, u, nil); err != nil {
					return err
				}
				// Avoid superfluous updates from converted zero defaults
				u.SetCreationTimestamp(metav1.Time{})

			}
		}
		return nil
	}
}

func replaceAnnotations(override *base.DeploymentOverride, deployment *appsv1.Deployment) {
	if deployment.GetAnnotations() == nil {
		deployment.Annotations = map[string]string{}
	}
	if deployment.Spec.Template.GetAnnotations() == nil {
		deployment.Spec.Template.Annotations = map[string]string{}
	}
	for key, val := range override.Annotations {
		deployment.Annotations[key] = val
		deployment.Spec.Template.Annotations[key] = val
	}
}

func replaceLabels(override *base.DeploymentOverride, deployment *appsv1.Deployment) {
	if deployment.GetLabels() == nil {
		deployment.Labels = map[string]string{}
	}
	if deployment.Spec.Template.GetLabels() == nil {
		deployment.Spec.Template.Labels = map[string]string{}
	}
	for key, val := range override.Labels {
		deployment.Labels[key] = val
		deployment.Spec.Template.Labels[key] = val
	}
}

func replaceReplicas(override *base.DeploymentOverride, deployment *appsv1.Deployment) {
	if override.Replicas != nil {
		deployment.Spec.Replicas = override.Replicas
	}
}

func replaceNodeSelector(override *base.DeploymentOverride, deployment *appsv1.Deployment) {
	if len(override.NodeSelector) > 0 {
		deployment.Spec.Template.Spec.NodeSelector = override.NodeSelector
	}
}

func replaceTolerations(override *base.DeploymentOverride, deployment *appsv1.Deployment) {
	if len(override.Tolerations) > 0 {
		deployment.Spec.Template.Spec.Tolerations = override.Tolerations
	}
}

func replaceAffinities(override *base.DeploymentOverride, deployment *appsv1.Deployment) {
	if override.Affinity != nil {
		deployment.Spec.Template.Spec.Affinity = override.Affinity
	}
}

func replaceResources(override *base.DeploymentOverride, deployment *appsv1.Deployment) {
	if len(override.Resources) > 0 {
		containers := deployment.Spec.Template.Spec.Containers
		for i := range containers {
			if override := find(override.Resources, containers[i].Name); override != nil {
				merge(&override.Limits, &containers[i].Resources.Limits)
				merge(&override.Requests, &containers[i].Resources.Requests)
			}
		}
	}
}
