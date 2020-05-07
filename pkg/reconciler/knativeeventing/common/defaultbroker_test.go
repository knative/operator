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

	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes/scheme"

	"knative.dev/operator/pkg/apis/operator/v1alpha1"
	util "knative.dev/operator/pkg/reconciler/common/testing"
)

var log = zap.NewNop().Sugar()

func TestDefaultBrokerTransform(t *testing.T) {
	tests := []struct {
		name               string
		configMap          corev1.ConfigMap
		defaultBrokerClass string
		expected           corev1.ConfigMap
	}{{
		name: "UsesDefaultWhenNotSpecified",
		configMap: makeConfigMap(t, "config-br-defaults", v1alpha1.ConfigMapData{
			"clusterDefault": {
				"brokerClass": "Foo",
				"apiVersion":  "v1",
				"kind":        "ConfigMap",
				"name":        "config-br-default-channel",
				"namespace":   "knative-eventing",
			},
		}),
		defaultBrokerClass: "",
		expected: makeConfigMap(t, "config-br-defaults", v1alpha1.ConfigMapData{
			"clusterDefault": {
				"brokerClass": "ChannelBasedBroker",
				"apiVersion":  "v1",
				"kind":        "ConfigMap",
				"name":        "config-br-default-channel",
				"namespace":   "knative-eventing",
			},
		}),
	}, {
		name: "UsesTheSpecifiedValueWhenSpecified",
		configMap: makeConfigMap(t, "config-br-defaults", v1alpha1.ConfigMapData{
			"clusterDefault": {
				"brokerClass": "Foo",
				"apiVersion":  "v1",
				"kind":        "ConfigMap",
				"name":        "config-br-default-channel",
				"namespace":   "knative-eventing",
			},
		}),
		defaultBrokerClass: "MyCustomerBroker",
		expected: makeConfigMap(t, "config-br-defaults", v1alpha1.ConfigMapData{
			"clusterDefault": {
				"brokerClass": "MyCustomerBroker",
				"apiVersion":  "v1",
				"kind":        "ConfigMap",
				"name":        "config-br-default-channel",
				"namespace":   "knative-eventing",
			},
		}),
	}, {
		name: "DoesNotTouchOtherConfigMaps",
		configMap: makeConfigMap(t, "some-other-config-map-foo-bar-baz", v1alpha1.ConfigMapData{
			"clusterDefault": {
				"brokerClass": "Foo",
				"apiVersion":  "v1",
				"kind":        "ConfigMap",
				"name":        "config-br-default-channel",
				"namespace":   "knative-eventing",
			},
		}),
		defaultBrokerClass: "MyCustomerBroker",
		expected: makeConfigMap(t, "config-br-defaults", v1alpha1.ConfigMapData{
			"clusterDefault": {
				"brokerClass": "Foo",
				"apiVersion":  "v1",
				"kind":        "ConfigMap",
				"name":        "config-br-default-channel",
				"namespace":   "knative-eventing",
			},
		}),
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unstructuredConfigMap := util.MakeUnstructured(t, &tt.configMap)
			instance := &v1alpha1.KnativeEventing{
				Spec: v1alpha1.KnativeEventingSpec{
					DefaultBrokerClass: tt.defaultBrokerClass,
				},
			}
			transform := DefaultBrokerConfigMapTransform(instance, log)
			transform(&unstructuredConfigMap)

			var configMap = &corev1.ConfigMap{}
			err := scheme.Scheme.Convert(&unstructuredConfigMap, configMap, nil)
			util.AssertEqual(t, err, nil)
			util.AssertDeepEqual(t, configMap.Data, tt.expected.Data)
		})
	}
}

func makeConfigMap(t *testing.T, name string, data v1alpha1.ConfigMapData) corev1.ConfigMap {
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
			"default-br-config": string(out),
		},
	}
}
