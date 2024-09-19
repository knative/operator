/*
Copyright 2023 The Knative Authors

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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/sets"
)

// When a Podspecable has HPA or a custom autoscaling, the replicas should be controlled by it instead of operator.
// Hence, skip changing the spec.replicas for these Podspecables.
func hasHorizontalPodOrCustomAutoscaler(name string) bool {
	return sets.NewString(
		"webhook",
		"activator",
		"3scale-kourier-gateway",
		"eventing-webhook",
		"mt-broker-ingress",
		"mt-broker-filter",
		"kafka-broker-dispatcher",
		"kafka-source-dispatcher",
		"kafka-channel-dispatcher",
	).Has(name)
}

// Maps a Podspecables name to the HPAs name.
// Add overrides here, if your HPA is named differently to the workloads name,
// if no override is defined, the name of the podspecable is used as HPA name.
func getHPAName(podspecableName string) string {
	overrides := map[string]string{
		"mt-broker-ingress": "broker-ingress-hpa",
		"mt-broker-filter":  "broker-filter-hpa",
	}
	if v, ok := overrides[podspecableName]; ok {
		return v
	} else {
		return podspecableName
	}
}

// hpaTransform sets the minReplicas and maxReplicas of an HPA based on a replica override value.
// If minReplica needs to be increased, the maxReplica is increased by the same value.
func hpaTransform(u *unstructured.Unstructured, replicas int64) error {
	if u.GetKind() != "HorizontalPodAutoscaler" {
		return nil
	}

	min, _, err := unstructured.NestedInt64(u.Object, "spec", "minReplicas")
	if err != nil {
		return err
	}

	// Do nothing if the HPA ships with even more replicas out of the box.
	if min >= replicas {
		return nil
	}

	if err := unstructured.SetNestedField(u.Object, replicas, "spec", "minReplicas"); err != nil {
		return err
	}

	max, found, err := unstructured.NestedInt64(u.Object, "spec", "maxReplicas")
	if err != nil {
		return err
	}

	// Do nothing if maxReplicas is not defined.
	if !found {
		return nil
	}

	// Increase maxReplicas to the amount that we increased,
	// because we need to avoid minReplicas > maxReplicas happening.
	if err := unstructured.SetNestedField(u.Object, max+(replicas-min), "spec", "maxReplicas"); err != nil {
		return err
	}
	return nil
}
