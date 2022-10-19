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
	"k8s.io/apimachinery/pkg/util/json"
	"knative.dev/operator/pkg/apis/operator/base"
)

func mergeProbe(override, tgt *v1.Probe) {
	if override == nil {
		return
	}
	var merged v1.Probe
	jtgt, _ := json.Marshal(*tgt)
	_ = json.Unmarshal(jtgt, &merged)
	jsrc, _ := json.Marshal(*override)
	_ = json.Unmarshal(jsrc, &merged)
	jmerged, _ := json.Marshal(merged)
	_ = json.Unmarshal(jmerged, tgt)
}

func findProbeOverride(probes []base.ProbesRequirementsOverride, name string) *base.ProbesRequirementsOverride {
	for _, override := range probes {
		if override.Container == name {
			return &override
		}
	}
	return nil
}
