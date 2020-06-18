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
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
)

func writeDefaultsToConfigMap(defaults interface{}, configMap *corev1.ConfigMap, key string, log *zap.SugaredLogger) error {
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

	configMap.Data[key] = string(yamlBytes)
	return nil
}
