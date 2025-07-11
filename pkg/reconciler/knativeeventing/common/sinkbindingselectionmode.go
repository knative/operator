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
	eventingv1beta1 "knative.dev/operator/pkg/apis/operator/v1beta1"
)

const SinkBindingSelectionModeEnvVarKey = "SINK_BINDING_SELECTION_MODE"

// SinkBindingSelectionModeTransform sets the eventing-webhook's SINK_BINDING_SELECTION_MODE env var to the value in the spec
func SinkBindingSelectionModeTransform(instance *eventingv1beta1.KnativeEventing, log *zap.SugaredLogger, convertFuncs ...func(in, out, context interface{}) error) mf.Transformer {
	var convert func(in, out, context interface{}) error
	if len(convertFuncs) > 0 && convertFuncs[0] != nil {
		convert = convertFuncs[0]
	} else {
		convert = scheme.Scheme.Convert
	}
	return func(u *unstructured.Unstructured) error {
		if u.GetKind() == "Deployment" && u.GetName() == "eventing-webhook" {
			deployment := &appsv1.Deployment{}
			err := convert(u, deployment, nil)
			if err != nil {
				log.Error(err, "Error converting Unstructured to Deployment", "unstructured", u, "deployment", deployment)
				return err
			}

			sinkBindingSelectionMode := instance.Spec.SinkBindingSelectionMode
			if sinkBindingSelectionMode == "" {
				if smFromWorkloadOverrides := sinkBindingSelectionModeFromWorkloadOverrides(instance); smFromWorkloadOverrides != "" {
					sinkBindingSelectionMode = smFromWorkloadOverrides
				} else {
					sinkBindingSelectionMode = "exclusion"
				}
			}

			for i := range deployment.Spec.Template.Spec.Containers {
				found := false
				c := &deployment.Spec.Template.Spec.Containers[i]
				for j := range c.Env {
					envVar := &c.Env[j]
					if envVar.Name == SinkBindingSelectionModeEnvVarKey {
						envVar.Value = sinkBindingSelectionMode
						found = true
						break
					}
				}
				if !found {
					c.Env = append(c.Env, corev1.EnvVar{Name: SinkBindingSelectionModeEnvVarKey, Value: sinkBindingSelectionMode})
				}
			}

			err = convert(deployment, u, nil)
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

func sinkBindingSelectionModeFromWorkloadOverrides(instance *eventingv1beta1.KnativeEventing) string {
	overrides := append(instance.Spec.Workloads, instance.Spec.DeploymentOverride...)
	for _, workloadOverride := range overrides {
		if workloadOverride.Name == "eventing-webhook" {
			for _, envRequirement := range workloadOverride.Env {
				if envRequirement.Container == "eventing-webhook" {
					for _, env := range envRequirement.EnvVars {
						if env.Name == SinkBindingSelectionModeEnvVarKey {
							return env.Value
						}
					}
				}
			}
		}
	}

	return ""
}
