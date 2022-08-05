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
	"context"
	"fmt"

	apix "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apixclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"knative.dev/pkg/apiextensions/storageversion"

	"knative.dev/pkg/logging"
)

var (
	OperatorCRDs = []string{"knativeservings.operator.knative.dev", "knativeeventings.operator.knative.dev"}
)

// MigrateCustomResource migrates the existing custom resource from v1alpha1 to v1beta1.
func MigrateCustomResource(ctx context.Context, dynamicClient dynamic.Interface, apixClient apixclient.Interface) error {
	logger := logging.FromContext(ctx)
	logger.Info("Migrating the existing custom resource")

	// Check if the existing CRD has v1alpha1 as one of the store versions. If so, migrate the existing
	// CR into the v1beta1 version; if not, no need to migrate.
	crdClient := apixClient.ApiextensionsV1().CustomResourceDefinitions()

	for _, crd := range OperatorCRDs {
		existingCRD, err := crdClient.Get(ctx, crd, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("Unable to fetch the CRD %s - %w", crd, err)
		}

		if existStorageVersion(existingCRD, "v1alpha1") {
			// Migrate the existing CR into the version v1beta1
			migrator := storageversion.NewMigrator(
				dynamicClient,
				apixClient,
			)

			gr := schema.ParseGroupResource(crd)
			if err = migrator.Migrate(ctx, gr); err != nil {
				if !apierrs.IsNotFound(err) {
					return fmt.Errorf("Unable to migrate the existing custom resource from v1alpha1 to v1beta1: %w", err)
				}
			} else {
				logger.Info("Successfully migrated the existing custom resource to v1beta1")
			}
		}
	}
	return nil
}

func existStorageVersion(crd *apix.CustomResourceDefinition, version string) bool {
	for _, v := range crd.Status.StoredVersions {
		if version == v {
			return true
		}
	}
	return false
}
