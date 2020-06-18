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
	"encoding/json"

	"github.com/ghodss/yaml"
	mf "github.com/manifestival/manifestival"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
	messagingconfig "knative.dev/eventing/pkg/apis/messaging/config"
	eventingv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
)

// DefaultChannelConfigMapTransform updates the default channel configMap with the value defined in the spec
func DefaultChannelConfigMapTransform(instance *eventingv1alpha1.KnativeEventing, log *zap.SugaredLogger) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if u.GetKind() == "ConfigMap" && u.GetName() == messagingconfig.ChannelDefaultsConfigName {
			var configMap = &corev1.ConfigMap{}
			err := scheme.Scheme.Convert(u, configMap, nil)
			if err != nil {
				log.Error(err, "Error converting Unstructured to ConfigMap", "unstructured", u, "configMap", configMap)
				return err
			}

			defaults, err := messagingconfig.NewChannelDefaultsConfigFromConfigMap(configMap)
			if err != nil {
				log.Error(err, "Error parsing default channel ConfigMap", "unstructured", u, "configMap", configMap)
				return err
			}

			defaultChannelTemplate := instance.Spec.DefaultChannelTemplate
			if defaultChannelTemplate == nil {
				defaultChannelTemplate = &messagingconfig.ChannelTemplateSpec{
					TypeMeta: metav1.TypeMeta{
						Kind:       "InMemoryChannel",
						APIVersion: "messaging.knative.dev/v1beta1",
					},
				}
			}
			defaults.ClusterDefault = defaultChannelTemplate

			err = writeChannelDefaultsToConfigMap(defaults, configMap, log)
			if err != nil {
				log.Error(err, "Error converting channel template defaults to default channel ConfigMap", "defaults", defaults, "configMap", configMap)
				return err
			}

			err = scheme.Scheme.Convert(configMap, u, nil)
			if err != nil {
				return err
			}
			// The zero-value timestamp defaulted by the conversion causes
			// superfluous updates
			u.SetCreationTimestamp(metav1.Time{})
			log.Debugw("Finished updating channel defaults configMap", "name", u.GetName(), "unstructured", u.Object)
		}
		return nil
	}
}

// TODO: make generic
func writeChannelDefaultsToConfigMap(defaults *messagingconfig.ChannelDefaults, configMap *corev1.ConfigMap, log *zap.SugaredLogger) error {
	jsonBytes, err := json.Marshal(defaults)
	if err != nil {
		log.Error("Defaults could not be converted to JSON", "defaults", defaults)
		return err
	}

	yamlBytes, err := yaml.JSONToYAML(jsonBytes)
	if err != nil {
		log.Error("Defaults could not be converted to YAML", "defaults", defaults)
		return err
	}

	configMap.Data[messagingconfig.ChannelDefaulterKey] = string(yamlBytes)
	return nil
}
