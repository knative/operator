/*
Copyright 2022 The Knative Authors

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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
	"knative.dev/operator/pkg/apis/operator/base"
)

// ServicesTransform transforms services based on the configuration in `spec.service`.
func ServicesTransform(obj base.KComponent, log *zap.SugaredLogger) mf.Transformer {
	overrides := obj.GetSpec().GetServiceOverride()
	if overrides == nil {
		return nil
	}
	return func(u *unstructured.Unstructured) error {
		for _, override := range overrides {
			if u.GetKind() == "Service" && u.GetName() == override.Name {
				service := &corev1.Service{}
				if err := scheme.Scheme.Convert(u, service, nil); err != nil {
					return err
				}
				overrideLabels(&override, service)
				overrideAnnotations(&override, service)
				overrideSelectors(&override, service)
				if err := scheme.Scheme.Convert(service, u, nil); err != nil {
					return err
				}
				// Avoid superfluous updates from converted zero defaults
				u.SetCreationTimestamp(metav1.Time{})
			}
		}
		return nil
	}
}

func overrideAnnotations(override *base.ServiceOverride, service *corev1.Service) {
	if service.GetAnnotations() == nil {
		service.Annotations = map[string]string{}
	}

	for key, val := range override.Annotations {
		service.Annotations[key] = val
	}
}

func overrideLabels(override *base.ServiceOverride, service *corev1.Service) {
	if service.GetLabels() == nil {
		service.Labels = map[string]string{}
	}

	for key, val := range override.Labels {
		service.Labels[key] = val
	}
}

func overrideSelectors(override *base.ServiceOverride, service *corev1.Service) {
	if service.Spec.Selector == nil {
		service.Spec.Selector = map[string]string{}
	}

	for key, val := range override.Selector {
		service.Spec.Selector[key] = val
	}
}
