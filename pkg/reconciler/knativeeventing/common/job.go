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
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
)

const (
	// StorageVersionMigration is the name of the job to be updated
	StorageVersionMigration = "storage-version-migration"

	// StorageVersionMigrationEventing is the name that the job name STORAGE_VERSION_MIGRATION will change into.
	StorageVersionMigrationEventing = "storage-version-migration-eventing"
)

// JobTransform updates the job with the expected value for the key app in the label
func JobTransform(log *zap.SugaredLogger) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if u.GetKind() == "Job" && u.GetName() == StorageVersionMigration {
			var job = &batchv1.Job{}
			err := scheme.Scheme.Convert(u, job, nil)
			if err != nil {
				log.Error(err, "Error converting Unstructured to Job", "unstructured", u, "job", job)
				return err
			}

			err = updateJobNameLabel(job)
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

func updateJobNameLabel(job *batchv1.Job) error {
	// Change the job name
	job.SetName(StorageVersionMigrationEventing)

	// Change the labels in metadata
	labels := job.GetLabels()
	labels["app"] = StorageVersionMigrationEventing
	job.SetLabels(labels)

	// Change the labels in spec.template.metadata
	labels = job.Spec.Template.GetLabels()
	labels["app"] = StorageVersionMigrationEventing
	job.Spec.Template.SetLabels(labels)

	return nil
}
