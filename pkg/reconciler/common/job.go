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
	"strings"

	mf "github.com/manifestival/manifestival"
	"go.uber.org/zap"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
)

// JobTransform updates the job with the expected value for the key app in the label
func JobTransform(obj v1alpha1.KComponent, log *zap.SugaredLogger) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if u.GetKind() == "Job" {
			var job = &batchv1.Job{}
			err := scheme.Scheme.Convert(u, job, nil)
			if err != nil {
				log.Error(err, "Error converting Unstructured to Job", "unstructured", u, "job", job)
				return err
			}

			err = updateJobName(obj, job)
			if err != nil {
				log.Error(err, "Error updating the job", "name", job.Name, "job", job)
				return err
			}

			err = scheme.Scheme.Convert(job, u, nil)
			if err != nil {
				return err
			}

			// The zero-value timestamp defaulted by the conversion causes superfluous updates
			u.SetCreationTimestamp(metav1.Time{})
			log.Debugw("Finished updating the job", "name", u.GetName(), "unstructured", u.Object)
		}

		return nil
	}
}

func updateJobName(instance v1alpha1.KComponent, job *batchv1.Job) error {
	component := ""
	switch instance.(type) {
	case *v1alpha1.KnativeServing:
		component = "serving"
	case *v1alpha1.KnativeEventing:
		component = "eventing"
	}
	version := TargetVersion(instance)

	// Change the job name
	jobName := job.GetName()
	if jobName == "" {
		jobName = job.GetGenerateName()
	}

	suffix := fmt.Sprintf("-%s-%s", component, version)

	if !strings.HasSuffix(jobName, suffix) {
		if strings.HasSuffix(jobName, component) {
			jobName = fmt.Sprintf("%s-%s", jobName, version)
		} else {
			jobName = fmt.Sprintf("%s%s", jobName, suffix)
		}
		job.SetName(jobName)
	}

	return nil
}
