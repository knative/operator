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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/sets"

	"knative.dev/operator/pkg/apis/operator/base"
)

func haUnSupported(name string) bool {
	return sets.NewString(
		"pingsource-mt-adapter",
	).Has(name)
}

// HighAvailabilityTransform mutates configmaps and replicacounts of certain
// controllers when HA control plane is specified.
func HighAvailabilityTransform(obj base.KComponent) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		// Use spec.deployments.replicas for the deployment instead of spec.high-availability.
		for _, override := range obj.GetSpec().GetWorkloadOverrides() {
			if override.Replicas != nil && override.Name == u.GetName() {
				return nil
			}
		}

		// stash the HA object
		ha := obj.GetSpec().GetHighAvailability()
		if ha == nil || ha.Replicas == nil {
			return nil
		}

		replicas := int64(*ha.Replicas)

		// Transform deployments that support HA.
		if u.GetKind() == "Deployment" && !haUnSupported(u.GetName()) && !hasHorizontalPodOrCustomAutoscaler(u.GetName()) {
			if err := unstructured.SetNestedField(u.Object, replicas, "spec", "replicas"); err != nil {
				return err
			}
		}

		if u.GetKind() == "HorizontalPodAutoscaler" {
			if err := hpaTransform(u, replicas); err != nil {
				return err
			}
		}

		return nil
	}
}
