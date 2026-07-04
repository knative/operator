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
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type unstructuredGetter interface {
	Get(obj *unstructured.Unstructured) (*unstructured.Unstructured, error)
}

// AggregationRuleTransform preserves the rules of aggregated ClusterRoles.
// The Kubernetes aggregation controller manages the rules field of aggregated
// ClusterRoles. Without this transform, manifestival overwrites the aggregated
// rules with the empty rules from the manifest, causing a race condition.
func AggregationRuleTransform(client unstructuredGetter) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if u.GetKind() == "ClusterRole" && u.Object["aggregationRule"] != nil {
			current, err := client.Get(u)
			if errors.IsNotFound(err) {
				return nil
			}
			if err != nil {
				return err
			}
			rules, found, err := unstructured.NestedSlice(current.Object, "rules")
			if err != nil {
				return err
			}
			if found {
				return unstructured.SetNestedSlice(u.Object, rules, "rules")
			}
		}
		return nil
	}
}
