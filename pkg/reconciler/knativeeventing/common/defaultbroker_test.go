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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"

	"knative.dev/operator/pkg/apis/operator/v1alpha1"
	util "knative.dev/operator/pkg/reconciler/common/testing"
)

func TestDefaultBrokerTransform(t *testing.T) {
	tests := []struct {
		name               string
		configMap          corev1.ConfigMap
		defaultBrokerClass string
		expected           corev1.ConfigMap
	}{{
		name: "UsesDefaultWhenNotSpecified",
		configMap: makeConfigMap(t, "config-br-defaults", "default-br-config", v1alpha1.ConfigMapData{
			"clusterDefault": {
				"brokerClass": "Foo",
				"apiVersion":  "v1",
				"kind":        "ConfigMap",
				"name":        "config-br-default-channel",
				"namespace":   "knative-eventing",
			},
		}),
		defaultBrokerClass: "",
		expected: makeConfigMap(t, "config-br-defaults", "default-br-config", v1alpha1.ConfigMapData{
			"clusterDefault": {
				"brokerClass": "MTChannelBasedBroker",
				"apiVersion":  "v1",
				"kind":        "ConfigMap",
				"name":        "config-br-default-channel",
				"namespace":   "knative-eventing",
			},
		}),
	}, {
		name: "UsesTheSpecifiedValueWhenSpecified",
		configMap: makeConfigMap(t, "config-br-defaults", "default-br-config", v1alpha1.ConfigMapData{
			"clusterDefault": {
				"brokerClass": "Foo",
				"apiVersion":  "v1",
				"kind":        "ConfigMap",
				"name":        "config-br-default-channel",
				"namespace":   "knative-eventing",
			},
		}),
		defaultBrokerClass: "MyCustomerBroker",
		expected: makeConfigMap(t, "config-br-defaults", "default-br-config", v1alpha1.ConfigMapData{
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
		configMap: makeConfigMap(t, "some-other-config-map-foo-bar-baz", "default-br-config", v1alpha1.ConfigMapData{
			"clusterDefault": {
				"brokerClass": "Foo",
				"apiVersion":  "v1",
				"kind":        "ConfigMap",
				"name":        "config-br-default-channel",
				"namespace":   "knative-eventing",
			},
		}),
		defaultBrokerClass: "MyCustomerBroker",
		expected: makeConfigMap(t, "config-br-defaults", "default-br-config", v1alpha1.ConfigMapData{
			"clusterDefault": {
				"brokerClass": "Foo",
				"apiVersion":  "v1",
				"kind":        "ConfigMap",
				"name":        "config-br-default-channel",
				"namespace":   "knative-eventing",
			},
		}),
	}}

	log := zap.NewNop().Sugar()

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
