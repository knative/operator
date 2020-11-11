// +build postdowngrade

/*
Copyright 2020 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e

import (
	"os"
	"testing"

	"knative.dev/operator/pkg/apis/operator/v1alpha1"

	"knative.dev/operator/pkg/reconciler/common"
	util "knative.dev/operator/pkg/reconciler/common/testing"
	"knative.dev/operator/test"
	"knative.dev/operator/test/client"
	"knative.dev/operator/test/resources"
)

// TestKnativeEventingPostDowngrade verifies the KnativeEventing creation, after downgraded to the previous version.
func TestKnativeEventingPostDowngrade(t *testing.T) {
	clients := client.Setup(t)

	names := test.ResourceNames{
		KnativeEventing: test.OperatorName,
		Namespace:       test.EventingOperatorNamespace,
	}

	// Create a KnativeEventing
	if _, err := resources.EnsureKnativeEventingExists(clients.KnativeEventing(), names); err != nil {
		t.Fatalf("KnativeService %q failed to create: %v", names.KnativeEventing, err)
	}

	// Verify if resources match the requirement for the previous release after downgrade
	t.Run("verify resources", func(t *testing.T) {
		resources.AssertKEOperatorCRReadyStatus(t, clients, names)
		resources.SetKodataDir()
		defer os.Unsetenv(common.KoEnvKey)

		_, err := common.TargetManifest(&v1alpha1.KnativeEventing{})
		if err != nil {
			t.Fatalf("Failed to get the manifest for Knative: %v", err)
		}

		instance := &v1alpha1.KnativeEventing{
			Spec: v1alpha1.KnativeEventingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Version: test.OperatorFlags.PreviousEventingVersion,
				},
			},
		}

		// Based on the previous release version, get the deployment resources.
		preManifest, err := common.TargetManifest(instance)
		if err != nil {
			t.Fatalf("Failed to get KnativeEventing manifest: %v", err)
		}
		expectedDeployments := resources.GetExpectedDeployments(preManifest)
		util.AssertEqual(t, len(expectedDeployments) > 0, true)
		resources.AssertKnativeDeploymentStatus(t, clients, names.Namespace, common.TargetVersion(instance),
			expectedDeployments)
	})
}

// TestKnativeServingPostDowngrade verifies the KnativeServing creation, after downgraded to the previous version.
func TestKnativeServingPostDowngrade(t *testing.T) {
	clients := client.Setup(t)

	names := test.ResourceNames{
		KnativeServing: test.OperatorName,
		Namespace:      test.ServingOperatorNamespace,
	}

	// Create a KnativeServing
	if _, err := resources.EnsureKnativeServingExists(clients.KnativeServing(), names); err != nil {
		t.Fatalf("KnativeService %q failed to create: %v", names.KnativeServing, err)
	}

	// Verify if resources match the requirement for the previous release after downgrade
	t.Run("verify resources", func(t *testing.T) {
		resources.AssertKSOperatorCRReadyStatus(t, clients, names)
		resources.SetKodataDir()
		defer os.Unsetenv(common.KoEnvKey)

		_, err := common.TargetManifest(&v1alpha1.KnativeServing{})
		if err != nil {
			t.Fatalf("Failed to get the manifest for Knative: %v", err)
		}

		instance := &v1alpha1.KnativeServing{
			Spec: v1alpha1.KnativeServingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Version: test.OperatorFlags.PreviousServingVersion,
				},
			},
		}

		// Based on the previous release version, get the deployment resources.
		preManifest, err := common.TargetManifest(instance)
		if err != nil {
			t.Fatalf("Failed to get KnativeServing manifest: %v", err)
		}
		expectedDeployments := resources.GetExpectedDeployments(preManifest)
		util.AssertEqual(t, len(expectedDeployments) > 0, true)
		resources.AssertKnativeDeploymentStatus(t, clients, names.Namespace, common.TargetVersion(instance),
			expectedDeployments)
	})
}
