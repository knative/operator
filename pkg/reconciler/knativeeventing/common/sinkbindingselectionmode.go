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
	eventingv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
)

// SinkBindingSelectionModeTransform updates the eventing-webhook's SINK_BINDING_SELECTION_MODE env var with the value in the spec
func SinkBindingSelectionModeTransform(instance *eventingv1alpha1.KnativeEventing, log *zap.SugaredLogger) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if u.GetKind() == "Deployment" && u.GetName() == "eventing-webhook" {
			var deployment = &appsv1.Deployment{}
			err := scheme.Scheme.Convert(u, deployment, nil)
			if err != nil {
				log.Error(err, "Error converting Unstructured to Deployment", "unstructured", u, "deployment", deployment)
				return err
			}

			sinkBindingSelectionMode := instance.Spec.SinkBindingSelectionMode
			if sinkBindingSelectionMode == "" {
				sinkBindingSelectionMode = "exclusion"
			}

			for i, _ := range deployment.Spec.Template.Spec.Containers {
				found := false
				c := &deployment.Spec.Template.Spec.Containers[i]
				for j, _ := range c.Env {
					envVar := &c.Env[j]
					if envVar.Name == "SINK_BINDING_SELECTION_MODE" {
						envVar.Value = sinkBindingSelectionMode
						found = true
						break
					}
				}
				if !found {
					c.Env = append(c.Env, corev1.EnvVar{Name: "SINK_BINDING_SELECTION_MODE", Value: sinkBindingSelectionMode})
				}
			}

			err = scheme.Scheme.Convert(deployment, u, nil)
			if err != nil {
				return err
			}
			// The zero-value timestamp defaulted by the conversion causes
			// superfluous updates
			u.SetCreationTimestamp(metav1.Time{})
			log.Debugw("Finished updating eventing-webhook deployment for sinkBindingSelectionMode", "name", u.GetName(), "unstructured", u.Object)
		}
		return nil
	}
}
