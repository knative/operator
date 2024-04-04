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
	"fmt"

	mf "github.com/manifestival/manifestival"
	"go.uber.org/zap"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"

	"knative.dev/operator/pkg/apis/operator/base"
)

// PodDisruptionBudgetsTransform transforms PodDisruptionBudgets based on the configuration in `spec.podDisruptionBudgets`.
func PodDisruptionBudgetsTransform(obj base.KComponent, log *zap.SugaredLogger) mf.Transformer {
	overrides := obj.GetSpec().GetPodDisruptionBudgetOverride()

	if overrides == nil {
		return nil
	}
	return func(u *unstructured.Unstructured) error {
		for _, override := range overrides {
			if u.GetKind() == "PodDisruptionBudget" && u.GetName() == override.Name {
				if override.MinAvailable == nil && override.MaxUnavailable == nil {
					return nil
				} else if override.MinAvailable != nil && override.MaxUnavailable != nil {
					return fmt.Errorf("both minAvailable and maxUnavailable are set for PodDisruptionBudget %s", override.Name)
				}

				podDisruptionBudget := &policyv1.PodDisruptionBudget{}
				if err := scheme.Scheme.Convert(u, podDisruptionBudget, nil); err != nil {
					return err
				}

				podDisruptionBudget.Spec.MinAvailable = override.MinAvailable
				podDisruptionBudget.Spec.MaxUnavailable = override.MaxUnavailable

				if err := scheme.Scheme.Convert(podDisruptionBudget, u, nil); err != nil {
					return err
				}

				// Avoid superfluous updates from converted zero defaults
				u.SetCreationTimestamp(metav1.Time{})
			}
		}
		return nil
	}
}
