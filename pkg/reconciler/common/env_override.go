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

package common

import (
	v1 "k8s.io/api/core/v1"
	"knative.dev/operator/pkg/apis/operator/base"
)

func mergeEnv(src, tgt *[]v1.EnvVar) {
	if len(*tgt) > 0 {
		for _, srcV := range *src {
			exists := false
			for i, tgtV := range *tgt {
				if srcV.Name == tgtV.Name {
					(*tgt)[i] = srcV
					exists = true
				}
			}
			if !exists {
				*tgt = append(*tgt, srcV)
			}
		}
	} else {
		*tgt = *src
	}
}

func findEnvOverride(resources []base.EnvRequirementsOverride, name string) *base.EnvRequirementsOverride {
	for _, override := range resources {
		if override.Container == name {
			return &override
		}
	}
	return nil
}
