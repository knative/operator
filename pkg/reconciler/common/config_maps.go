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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"knative.dev/operator/pkg/apis/operator/base"
)

// ConfigMapTransform updates the ConfigMap with the values specified in operator CR
func ConfigMapTransform(config base.ConfigMapData, log *zap.SugaredLogger) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		// Let any config in instance override everything else
		if u.GetKind() == "ConfigMap" {
			if data, ok := config[u.GetName()]; ok {
				return UpdateConfigMap(u, data, log)
			}
			// The "config-" prefix is optional
			if data, ok := config[u.GetName()[len(`config-`):]]; ok {
				return UpdateConfigMap(u, data, log)
			}
		}
		return nil
	}
}

// UpdateConfigMap set some data in a configmap, only overwriting common keys if they differ
func UpdateConfigMap(cm *unstructured.Unstructured, data map[string]string, log *zap.SugaredLogger) error {
	for k, v := range data {
		message := []interface{}{"map", cm.GetName(), k, v}
		x, found, err := unstructured.NestedFieldNoCopy(cm.Object, "data", k)
		if err != nil {
			return err
		}
		if found {
			if v == x {
				continue
			}
			message = append(message, "previous", x)
		}
		log.Infow("Setting", message...)
		if err := unstructured.SetNestedField(cm.Object, v, "data", k); err != nil {
			return err
		}
	}
	return nil
}
