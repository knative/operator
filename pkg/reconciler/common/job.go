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
	"fmt"

	mf "github.com/manifestival/manifestival"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
)

// JobTransform updates the job with the expected value for the key app in the label
func JobTransform(obj v1alpha1.KComponent) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if u.GetKind() == "Job" {
			component := "serving"
			if _, ok := obj.(*v1alpha1.KnativeEventing); ok {
				component = "eventing"
			}
			if u.GetName() == "" {
				u.SetName(fmt.Sprintf("%s%s-%s", u.GetGenerateName(), component, TargetVersion(obj)))
			} else {
				u.SetName(fmt.Sprintf("%s-%s-%s", u.GetName(), component, TargetVersion(obj)))
			}
		}
		return nil
	}
}
