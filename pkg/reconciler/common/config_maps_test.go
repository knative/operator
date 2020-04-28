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

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	util "knative.dev/operator/pkg/reconciler/common/testing"
)

type configMapData struct {
	name string
	data map[string]string
}

type updateConfigMapTest struct {
	name      string
	config    configMapData
	configMap corev1.ConfigMap
	expected  corev1.ConfigMap
}

func makeconfigMapData(name string, data map[string]string) configMapData {
	return configMapData{
		name: name,
		data: data,
	}
}

func createConfigMapTests(t *testing.T) []updateConfigMapTest {
	return []updateConfigMapTest{
		{
			name: "change-config-logging",
			config: makeconfigMapData("logging", map[string]string{
				"loglevel.controller": "debug",
				"loglevel.webhook":    "debug",
			}),
			configMap: createConfigMap("config-logging", map[string]string{
				"loglevel.controller": "info",
				"loglevel.webhook":    "info",
			}),
			expected: createConfigMap("config-logging", map[string]string{
				"loglevel.controller": "debug",
				"loglevel.webhook":    "debug",
			}),
		},
		{
			name: "change-config-logging-empty-data",
			config: makeconfigMapData("logging", map[string]string{
				"loglevel.controller": "debug",
				"loglevel.webhook":    "debug",
			}),
			configMap: createConfigMap("config-logging", nil),
			expected: createConfigMap("config-logging", map[string]string{
				"loglevel.controller": "debug",
				"loglevel.webhook":    "debug",
			}),
		},
	}
}

func createConfigMap(name string, data map[string]string) corev1.ConfigMap {
	return corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind: "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Data: data,
	}
}

func TestConfigMapTransform(t *testing.T) {
	for _, tt := range createConfigMapTests(t) {
		t.Run(tt.name, func(t *testing.T) {
			runConfigMapTransformTest(t, &tt)
		})
	}
}

func runConfigMapTransformTest(t *testing.T, tt *updateConfigMapTest) {
	unstructuredConfigMap := util.MakeUnstructured(t, &tt.configMap)
	config := map[string]map[string]string{
		tt.config.name: tt.config.data,
	}
	configMapTransform := ConfigMapTransform(config, log)
	configMapTransform(&unstructuredConfigMap)
	validateConfigMapChanged(t, tt, &unstructuredConfigMap)
}

func validateConfigMapChanged(t *testing.T, tt *updateConfigMapTest, u *unstructured.Unstructured) {
	var configMap = &corev1.ConfigMap{}
	err := scheme.Scheme.Convert(u, configMap, nil)
	util.AssertEqual(t, err, nil)
	util.AssertDeepEqual(t, configMap.Data, tt.expected.Data)
}
