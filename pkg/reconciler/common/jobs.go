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
)

// JobSidecarTransform updates the Job with the annotation sidecar.istio.io/inject equal to false
func JobSidecarTransform() mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if u.GetKind() == "Job" {
			annotationData := u.GetAnnotations()
			if annotationData == nil {
				annotationData = make(map[string]string)
			}
			annotationData["sidecar.istio.io/inject"] = "false"
			return UpdateJob(u, annotationData)
		}
		return nil
	}
}

// UpdateJob sets the annotation of the Job
func UpdateJob(job *unstructured.Unstructured, annotationData map[string]string) error {
	job.SetAnnotations(annotationData)
	return nil
}
