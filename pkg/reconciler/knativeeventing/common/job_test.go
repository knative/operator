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
	"testing"

	"go.uber.org/zap"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"

	util "knative.dev/operator/pkg/reconciler/common/testing"
)

func TestJobTransform(t *testing.T) {
	tests := []struct {
		name     string
		job      batchv1.Job
		expected batchv1.Job
	}{{
		name:     "ChangeNameAndLabels",
		job:      makeJob(STORAGE_VERSION_MIGRATION),
		expected: makeJob(STORAGE_VERSION_MIGRATION_EVENTING),
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unstructuredJob := util.MakeUnstructured(t, &tt.job)
			transform := JobTransform(zap.NewNop().Sugar())
			transform(&unstructuredJob)

			var job = &batchv1.Job{}
			err := scheme.Scheme.Convert(&unstructuredJob, job, nil)
			util.AssertEqual(t, err, nil)
			util.AssertDeepEqual(t, job.GetObjectMeta(), tt.expected.GetObjectMeta())
			util.AssertDeepEqual(t, job.Spec, tt.expected.Spec)
		})
	}
}

func makeJob(name string) batchv1.Job {
	return batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Job",
			APIVersion: "batch/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{"app": name},
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": name},
				},
			},
		},
	}
}
