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
	mf "github.com/manifestival/manifestival/pkg/transform"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
)

// ResourceRequirementsTransform configures the resource requests for
// all containers within all deployments in the manifest
func ResourceRequirementsTransform(resources []v1alpha1.ResourceRequirementsOverride, log *zap.SugaredLogger) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if u.GetKind() == "Deployment" {
			deployment := &appsv1.Deployment{}
			if err := scheme.Scheme.Convert(u, deployment, nil); err != nil {
				return err
			}
			containers := deployment.Spec.Template.Spec.Containers
			for i := range containers {
				if override := find(resources, containers[i].Name); override != nil {
					merge(&override.Limits, &containers[i].Resources.Limits)
					merge(&override.Requests, &containers[i].Resources.Requests)
				}
			}
			if err := scheme.Scheme.Convert(deployment, u, nil); err != nil {
				return err
			}
			// Avoid superfluous updates from converted zero defaults
			u.SetCreationTimestamp(metav1.Time{})
		}
		return nil
	}
}

func merge(src, tgt *v1.ResourceList) {
	if len(*tgt) > 0 {
		for k, v := range *src {
			(*tgt)[k] = v
		}
	} else {
		*tgt = *src
	}
}

func find(resources []v1alpha1.ResourceRequirementsOverride, name string) *v1alpha1.ResourceRequirementsOverride {
	for _, override := range resources {
		if override.Container == name {
			return &override
		}
	}
	return nil
}
