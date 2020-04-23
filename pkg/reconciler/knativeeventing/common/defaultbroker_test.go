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

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"k8s.io/client-go/kubernetes/scheme"

	eventingv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
	util "knative.dev/operator/pkg/reconciler/common/testing"
)

var log = zap.NewNop().Sugar()

type updateDefaultBrokerTest struct {
	t                  *testing.T
	name               string
	configMap          corev1.ConfigMap
	defaultBrokerClass string
	expected           corev1.ConfigMap
}

func createupdateDefaultBrokerTests(t *testing.T) []updateDefaultBrokerTest {
	return []updateDefaultBrokerTest{
		{
			name: "UsesDefaultWhenNotSpecified",
			configMap: makeConfigMap(t, "config-br-defaults", map[string]map[string]string{
				"clusterDefault": {
					"brokerClass": "Foo",
					"apiVersion":  "v1",
					"kind":        "ConfigMap",
					"name":        "config-br-default-channel",
					"namespace":   "knative-eventing",
				},
			}),
			defaultBrokerClass: "",
			expected: makeConfigMap(t, "config-br-defaults", map[string]map[string]string{
				"clusterDefault": {
					"brokerClass": "ChannelBasedBroker",
					"apiVersion":  "v1",
					"kind":        "ConfigMap",
					"name":        "config-br-default-channel",
					"namespace":   "knative-eventing",
				},
			}),
		},
		{
			name: "UsesTheSpecifiedValueWhenSpecified",
			configMap: makeConfigMap(t, "config-br-defaults", map[string]map[string]string{
				"clusterDefault": {
					"brokerClass": "Foo",
					"apiVersion":  "v1",
					"kind":        "ConfigMap",
					"name":        "config-br-default-channel",
					"namespace":   "knative-eventing",
				},
			}),
			defaultBrokerClass: "MyCustomerBroker",
			expected: makeConfigMap(t, "config-br-defaults", map[string]map[string]string{
				"clusterDefault": {
					"brokerClass": "MyCustomerBroker",
					"apiVersion":  "v1",
					"kind":        "ConfigMap",
					"name":        "config-br-default-channel",
					"namespace":   "knative-eventing",
				},
			}),
		},
		{
			name: "DoesNotTouchOtherConfigMaps",
			configMap: makeConfigMap(t, "some-other-config-map-foo-bar-baz", map[string]map[string]string{
				"clusterDefault": {
					"brokerClass": "Foo",
					"apiVersion":  "v1",
					"kind":        "ConfigMap",
					"name":        "config-br-default-channel",
					"namespace":   "knative-eventing",
				},
			}),
			defaultBrokerClass: "MyCustomerBroker",
			expected: makeConfigMap(t, "config-br-defaults", map[string]map[string]string{
				"clusterDefault": {
					"brokerClass": "Foo",
					"apiVersion":  "v1",
					"kind":        "ConfigMap",
					"name":        "config-br-default-channel",
					"namespace":   "knative-eventing",
				},
			}),
		},
	}
}

func TestDefaultBrokerTransform(t *testing.T) {
	updateDefaultBrokerTests := createupdateDefaultBrokerTests(t)

	for _, tt := range updateDefaultBrokerTests {
		t.Run(tt.name, func(t *testing.T) {
			runDefaultImageBrokerTransformTest(t, &tt)
		})
	}
}

func runDefaultImageBrokerTransformTest(t *testing.T, tt *updateDefaultBrokerTest) {
	unstructuredConfigMap := util.MakeUnstructured(t, &tt.configMap)
	instance := &eventingv1alpha1.KnativeEventing{
		Spec: eventingv1alpha1.KnativeEventingSpec{
			DefaultBrokerClass: tt.defaultBrokerClass,
		},
	}
	transform := DefaultBrokerConfigMapTransform(instance, log)
	transform(&unstructuredConfigMap)
	validateUnstructedConfigMapChanged(t, tt, &unstructuredConfigMap)
}

func validateUnstructedConfigMapChanged(t *testing.T, tt *updateDefaultBrokerTest, u *unstructured.Unstructured) {
	var configMap = &corev1.ConfigMap{}
	err := scheme.Scheme.Convert(u, configMap, nil)
	util.AssertEqual(t, err, nil)
	util.AssertDeepEqual(t, configMap.Data, tt.expected.Data)
}

func makeConfigMap(t *testing.T, name string, data map[string]map[string]string) corev1.ConfigMap {
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
