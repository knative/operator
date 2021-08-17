//go:build e2e
// +build e2e

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
	"testing"

	"knative.dev/operator/test"
	"knative.dev/operator/test/client"
	"knative.dev/operator/test/resources"
)

// TestKnativeServingDeployment verifies the KnativeServing creation, deployment recreation, and KnativeServing deletion.
func TestKnativeServingDeployment(t *testing.T) {
	clients := client.Setup(t)

	names := test.ResourceNames{
		KnativeServing: test.OperatorName,
		Namespace:      test.ServingOperatorNamespace,
	}

	test.CleanupOnInterrupt(func() { test.TearDown(clients, names) })
	defer test.TearDown(clients, names)

	// Create a KnativeServing
	if _, err := resources.EnsureKnativeServingExists(clients.KnativeServing(), names); err != nil {
		t.Fatalf("KnativeService %q failed to create: %v", names.KnativeServing, err)
	}

	// Test if KnativeServing can reach the READY status
	t.Run("create", func(t *testing.T) {
		resources.AssertKSOperatorCRReadyStatus(t, clients, names)
	})

	t.Run("configure", func(t *testing.T) {
		resources.AssertKSOperatorCRReadyStatus(t, clients, names)
		resources.KSOperatorCRVerifyConfiguration(t, clients, names)
	})

	// Delete the deployments one by one to see if they will be recreated.
	t.Run("restore", func(t *testing.T) {
		resources.AssertKSOperatorCRReadyStatus(t, clients, names)
		resources.DeleteAndVerifyDeployments(t, clients, names)
	})

	// Delete the KnativeServing to see if all resources will be removed
	t.Run("delete", func(t *testing.T) {
		resources.AssertKSOperatorCRReadyStatus(t, clients, names)
		resources.KSOperatorCRDelete(t, clients, names)
	})
}
