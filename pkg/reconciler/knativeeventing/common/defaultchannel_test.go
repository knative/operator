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

	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes/scheme"

	messagingconfig "knative.dev/eventing/pkg/apis/messaging/config"
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
	util "knative.dev/operator/pkg/reconciler/common/testing"
)

// TODO: common
//var log = zap.NewNop().Sugar()

func TestDefaultChannelTemplateTransform(t *testing.T) {
	tests := []struct {
		name                   string
		configMap              corev1.ConfigMap
		defaultChannelTemplate *messagingconfig.ChannelTemplateSpec
		expected               corev1.ConfigMap
	}{{
		name: "UsesDefaultWhenNotSpecified",
		configMap: makeDefaultChannelConfigMap(t, "default-ch-webhook", v1alpha1.ConfigMapData{
			"clusterDefault": {
				"apiVersion": "to-be-overridden-api-version",
				"kind":       "to-be-overridden-kind",
			},
		}),
		defaultChannelTemplate: nil,
		expected: makeDefaultChannelConfigMap(t, "default-ch-webhook", v1alpha1.ConfigMapData{
			"clusterDefault": {
				"apiVersion": "messaging.knative.dev/v1beta1",
				"kind":       "InMemoryChannel",
			},
		}),
	}, {
		name: "UsesTheSpecifiedValueWhenSpecified",
		configMap: makeDefaultChannelConfigMap(t, "default-ch-webhook", v1alpha1.ConfigMapData{
			"clusterDefault": {
				"apiVersion": "to-be-overridden-api-version",
				"kind":       "to-be-overridden-kind",
			},
		}),
		defaultChannelTemplate: &messagingconfig.ChannelTemplateSpec{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "example.org/v1beta1",
				Kind:       "CustomChannel",
			},
		},
		expected: makeDefaultChannelConfigMap(t, "default-ch-webhook", v1alpha1.ConfigMapData{
			"clusterDefault": {
				"apiVersion": "example.org/v1beta1",
				"kind":       "CustomChannel",
			},
		}),
	}, {
		name: "DoesNotTouchOtherConfigMaps",
		configMap: makeDefaultChannelConfigMap(t, "some-other-config-map-foo-bar-baz", v1alpha1.ConfigMapData{
			"clusterDefault": {
				"apiVersion": "to-be-overridden-api-version",
				"kind":       "to-be-overridden-kind",
			},
		}),
		defaultChannelTemplate: &messagingconfig.ChannelTemplateSpec{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "example.org/v1beta1",
				Kind:       "CustomChannel",
			},
		},
		expected: makeDefaultChannelConfigMap(t, "default-ch-webhook", v1alpha1.ConfigMapData{
			"clusterDefault": {
				"apiVersion": "to-be-overridden-api-version",
				"kind":       "to-be-overridden-kind",
			},
		}),
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unstructuredConfigMap := util.MakeUnstructured(t, &tt.configMap)
			instance := &v1alpha1.KnativeEventing{
				Spec: v1alpha1.KnativeEventingSpec{
					DefaultChannelTemplate: tt.defaultChannelTemplate,
				},
			}
			transform := DefaultChannelConfigMapTransform(instance, log)
			transform(&unstructuredConfigMap)

			var configMap = &corev1.ConfigMap{}
			err := scheme.Scheme.Convert(&unstructuredConfigMap, configMap, nil)
			util.AssertEqual(t, err, nil)
			util.AssertDeepEqual(t, configMap.Data, tt.expected.Data)
		})
	}
}

// TODO: merge shit
func makeDefaultChannelConfigMap(t *testing.T, name string, data v1alpha1.ConfigMapData) corev1.ConfigMap {
	out, err := yaml.Marshal(&data)
	if err != nil {
		t.Fatal("Unable to marshal test data. Possible implementation problem.", "data", data)
	}
	return corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind: "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Data: map[string]string{
			"default-ch-config": string(out),
		},
	}
}
