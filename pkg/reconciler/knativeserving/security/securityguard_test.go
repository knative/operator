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

package security

import (
	"os"
	"testing"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"knative.dev/operator/pkg/apis/operator/base"
	"knative.dev/operator/pkg/apis/operator/v1beta1"
	servingv1beta1 "knative.dev/operator/pkg/apis/operator/v1beta1"
	"knative.dev/operator/pkg/reconciler/common"
	util "knative.dev/operator/pkg/reconciler/common/testing"
)

var log = zap.NewNop().Sugar()

func TestConfigMapTransform(t *testing.T) {
	os.Setenv(common.KoEnvKey, "testdata/kodata")
	defer os.Unsetenv(common.KoEnvKey)

	tests := []struct {
		name      string
		configMap corev1.ConfigMap
		instance  *v1beta1.KnativeServing
		expected  corev1.ConfigMap
	}{{
		name: "Update the ConfigMap config-features",
		configMap: corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				Kind: "ConfigMap",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "config-features",
			},
			Data: map[string]string{},
		},
		instance: &v1beta1.KnativeServing{
			Spec: servingv1beta1.KnativeServingSpec{
				CommonSpec: base.CommonSpec{
					Version: "1.8.0",
				},
				Security: &servingv1beta1.SecurityConfigs{
					SecurityGuard: base.SecurityGuardConfiguration{
						Enabled: true,
					},
				},
			},
		},
		expected: corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				Kind: "ConfigMap",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "config-features",
			},
			Data: map[string]string{
				QueueProxyMountPodInfoKey: "allowed",
			},
		},
	}, {
		name: "Do not update the ConfigMap config-features",
		configMap: corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				Kind: "ConfigMap",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "config-features",
			},
			Data: map[string]string{},
		},
		instance: &v1beta1.KnativeServing{
			Spec: servingv1beta1.KnativeServingSpec{
				CommonSpec: base.CommonSpec{
					Version: "1.8.0",
				},
				Security: &servingv1beta1.SecurityConfigs{
					SecurityGuard: base.SecurityGuardConfiguration{
						Enabled: false,
					},
				},
			},
		},
		expected: corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				Kind: "ConfigMap",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "config-features",
			},
		},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unstructuredConfigMap := util.MakeUnstructured(t, &tt.configMap)
			transform := configMapTransform(tt.instance, log)
			transform(&unstructuredConfigMap)

			var configMap = &corev1.ConfigMap{}
			err := scheme.Scheme.Convert(&unstructuredConfigMap, configMap, nil)
			util.AssertEqual(t, err, nil)
			util.AssertDeepEqual(t, configMap.Data, tt.expected.Data)
		})
	}
}
