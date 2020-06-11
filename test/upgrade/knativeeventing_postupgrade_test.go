// +build postupgrade

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

	mf "github.com/manifestival/manifestival"

	"knative.dev/operator/pkg/reconciler/common"
	util "knative.dev/operator/pkg/reconciler/common/testing"
	"knative.dev/operator/test"
	"knative.dev/operator/test/client"
	"knative.dev/operator/test/resources"
	"knative.dev/pkg/test/logstream"
)

// TestKnativeEventingUpgrade verifies the KnativeEventing creation, deployment recreation, and KnativeEventing deletion
// after upgraded to the latest HEAD at master, with the latest generated manifest of KnativeEventing.
func TestKnativeEventingUpgrade(t *testing.T) {
	cancel := logstream.Start(t)
	defer cancel()
	clients := client.Setup(t)

	names := test.ResourceNames{
		KnativeEventing: test.OperatorName,
		Namespace:       test.EventingOperatorNamespace,
	}

	// Create a KnativeEventing
	if _, err := resources.EnsureKnativeEventingExists(clients.KnativeEventing(), names); err != nil {
		t.Fatalf("KnativeService %q failed to create: %v", names.KnativeEventing, err)
	}

	// Verify if resources match the requirement for the previous release before upgrade
	t.Run("verify resources", func(t *testing.T) {
		resources.AssertKEOperatorCRReadyStatus(t, clients, names)
		kcomponent := "knative-eventing"
		resources.SetKodataDir()
		defer os.Unsetenv(common.KoEnvKey)
		version := common.LatestRelease(kcomponent)
		// Based on the latest release version, get the deployment resources.
		targetManifest, expectedDeployments := resources.GetExpectedDeployments(t, version, kcomponent)
		util.AssertEqual(t, len(expectedDeployments) > 0, true)
		resources.AssertKnativeDeploymentStatus(t, clients, names.Namespace, expectedDeployments)

		preEventingVer := test.OperatorFlags.PreviousEventingVersion
		if preEventingVer == "" {
			preEventingVer = version
		}
		// Compare the previous manifest with the target manifest, we verify that all the obsolete resources
		// do not exist any more.
		preManifest, err := resources.GetManifest(preEventingVer, kcomponent)
		if err != nil {
			t.Fatalf("Failed to get KnativeEventing manifest: %v", err)
		}
		resources.AssertKnativeObsoleteResource(t, clients, names.Namespace,
			preManifest.Filter(mf.None(mf.In(targetManifest))).Resources())
	})
}
