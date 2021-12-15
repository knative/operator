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

	"knative.dev/operator/pkg/apis/operator/v1alpha1"

	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"

	util "knative.dev/operator/pkg/reconciler/common/testing"
)

const (
	StorageVersionMigration = "storage-version-migration"
)

func TestJobTransform(t *testing.T) {
	tests := []struct {
		name      string
		component v1alpha1.KComponent
		job       batchv1.Job
		expected  string
	}{{
		name: "ChangeNameForServingJob",
		component: &v1alpha1.KnativeServing{
			Spec: v1alpha1.KnativeServingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Version: "0.15.2",
				},
			}},
		job:      createJob(StorageVersionMigration, ""),
		expected: StorageVersionMigration + "-serving-0.15.2",
	}, {
		name: "ChangeNameForEventingJob",
		component: &v1alpha1.KnativeEventing{
			Spec: v1alpha1.KnativeEventingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Version: "0.16.0",
				},
			}},
		job:      createJob(StorageVersionMigration, ""),
		expected: StorageVersionMigration + "-eventing-0.16.0",
	}, {
		name: "ChangeNameWithGeneratedNameForServingJob",
		component: &v1alpha1.KnativeServing{
			Spec: v1alpha1.KnativeServingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Version: "0.15.2",
				},
			}},
		job:      createJob("", StorageVersionMigration),
		expected: StorageVersionMigration + "-serving-0.15.2",
	}, {
		name: "ChangeNameWithGeneratedNameForEventingJob",
		component: &v1alpha1.KnativeEventing{
			Spec: v1alpha1.KnativeEventingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Version: "0.16.0",
				},
			}},
		job:      createJob("", StorageVersionMigration),
		expected: StorageVersionMigration + "-eventing-0.16.0",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unstructuredJob := util.MakeUnstructured(t, &tt.job)
			transform := JobTransform(tt.component)
			transform(&unstructuredJob)

			var job = &batchv1.Job{}
			err := scheme.Scheme.Convert(&unstructuredJob, job, nil)
			util.AssertEqual(t, err, nil)
			util.AssertDeepEqual(t, job.Name, tt.expected)
		})
	}
}

func createJob(name, gen string) batchv1.Job {
	return batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Job",
			APIVersion: "batch/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:         name,
			GenerateName: gen + "-",
		},
	}
}
