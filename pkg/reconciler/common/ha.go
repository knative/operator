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
	"k8s.io/apimachinery/pkg/util/sets"

	"knative.dev/operator/pkg/apis/operator/base"
)

func haUnSupported(obj base.KComponent) sets.String {
	return sets.NewString(
		"pingsource-mt-adapter",
	)
}

// When Deployment has HPA, the replicas should be controlled by HPA's minReplicas instead of operator.
// Hence, skip changing the spec.replicas in deployment directory for these Deployments.
func hasHorizontalPodAutoscaler(obj base.KComponent) sets.String {
	return sets.NewString(
		"webhook",
		"activator",
	)
}

// HighAvailabilityTransform mutates configmaps and replicacounts of certain
// controllers when HA control plane is specified.
func HighAvailabilityTransform(obj base.KComponent, log *zap.SugaredLogger) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		// Use spec.deployments.replicas for the deployment instead of spec.high-availability.
		for _, override := range obj.GetSpec().GetOverrides() {
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
		if u.GetKind() == "Deployment" && !haUnSupported(obj).Has(u.GetName()) && !hasHorizontalPodAutoscaler(obj).Has(u.GetName()) {
			if err := unstructured.SetNestedField(u.Object, replicas, "spec", "replicas"); err != nil {
				return err
			}
		}

		if u.GetKind() == "HorizontalPodAutoscaler" {
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
			// because we need to avoid minReplicas > maxReplicas happenning.
			if err := unstructured.SetNestedField(u.Object, max+(replicas-min), "spec", "maxReplicas"); err != nil {
				return err
			}
		}

		return nil
	}
}
