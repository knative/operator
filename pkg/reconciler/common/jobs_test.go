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

	batchv1 "k8s.io/api/batch/v1"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	util "knative.dev/operator/pkg/reconciler/common/testing"
)

type jobData struct {
	name        string
	annotations map[string]string
}

type updateJobTest struct {
	name     string
	job      batchv1.Job
	expected batchv1.Job
}

func makeJobData(name string, data map[string]string) jobData {
	return jobData{
		name:        name,
		annotations: data,
	}
}

func createJobTests(t *testing.T) []updateJobTest {
	return []updateJobTest{
		{
			name: "non-empty-annotation",
			job: createJob("config-logging", map[string]string{
				"loglevel.controller":     "info",
				"loglevel.webhook":        "info",
				"sidecar.istio.io/inject": "true",
			}),
			expected: createJob("config-logging", map[string]string{
				"loglevel.controller":     "info",
				"loglevel.webhook":        "info",
				"sidecar.istio.io/inject": "false",
			}),
		},
		{
			name: "empty-annotation",
			job:  createJob("config-logging", nil),
			expected: createJob("config-logging", map[string]string{
				"sidecar.istio.io/inject": "false",
			}),
		},
		{
			name: "change-using-real-configmap-name",
			job: createJob("config-logging", map[string]string{
				"loglevel.controller": "info",
				"loglevel.webhook":    "info",
			}),
			expected: createJob("config-logging", map[string]string{
				"loglevel.controller":     "info",
				"loglevel.webhook":        "info",
				"sidecar.istio.io/inject": "false",
			}),
		},
	}
}

func createJob(name string, annotationData map[string]string) batchv1.Job {
	return batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			Kind: "Job",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Annotations: annotationData,
		},
	}
}

func TestJobTransform(t *testing.T) {
	for _, tt := range createJobTests(t) {
		t.Run(tt.name, func(t *testing.T) {
			runJobTransformTest(t, &tt)
		})
	}
}

func runJobTransformTest(t *testing.T, tt *updateJobTest) {
	unstructuredJob := util.MakeUnstructured(t, &tt.job)
	configMapTransform := JobSidecarTransform()
	configMapTransform(&unstructuredJob)
	validateJobChanged(t, tt, &unstructuredJob)
}

func validateJobChanged(t *testing.T, tt *updateJobTest, u *unstructured.Unstructured) {
	var job = &batchv1.Job{}
	err := scheme.Scheme.Convert(u, job, nil)
	util.AssertEqual(t, err, nil)
	util.AssertDeepEqual(t, job.Annotations, tt.expected.Annotations)
}
