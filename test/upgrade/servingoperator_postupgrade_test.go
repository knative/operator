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

	util "knative.dev/operator/pkg/reconciler/common/testing"

	mf "github.com/manifestival/manifestival"
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
	"knative.dev/operator/pkg/reconciler/common"
	"knative.dev/operator/test"
	"knative.dev/operator/test/client"
	"knative.dev/operator/test/resources"
)

// TestKnativeServingPostUpgrade verifies the KnativeServing creation, deployment recreation, and KnativeServing deletion
// after the operator upgrades with the latest generated manifest of Knative Serving.
func TestKnativeServingPostUpgrade(t *testing.T) {
	clients := client.Setup(t)

	names := test.ResourceNames{
		KnativeServing: test.OperatorName,
		Namespace:      test.ServingOperatorNamespace,
	}

	// Create a KnativeServing custom resource, if it does not exist
	if _, err := resources.EnsureKnativeServingExists(clients.KnativeServing(), names); err != nil {
		t.Fatalf("KnativeService %q failed to create: %v", names.KnativeServing, err)
	}

	// Verify if resources match the latest requirement after upgrade
	t.Run("verify resources", func(t *testing.T) {
		// TODO: We only verify the deployment, but we need to add other resources as well, like ServiceAccount, ClusterRoleBinding, etc.
		resources.SetKodataDir()
		defer os.Unsetenv(common.KoEnvKey)
		targetManifest, expectedDeployments := resources.GetExpectedDeployments(t, &v1alpha1.KnativeServing{})
		util.AssertEqual(t, len(expectedDeployments) > 0, true)
		resources.AssertKnativeDeploymentStatus(t, clients, names.Namespace, expectedDeployments)
		resources.AssertKSOperatorCRReadyStatus(t, clients, names)

		instance := &v1alpha1.KnativeServing{
			Spec: v1alpha1.KnativeServingSpec{
				CommonSpec: v1alpha1.CommonSpec{
					Version: test.OperatorFlags.PreviousServingVersion,
				},
			},
		}
		// Compare the previous manifest with the target manifest, we verify that all the obsolete resources
		// do not exist any more.
		preManifest, err := common.TargetManifest(instance)
		if err != nil {
			t.Fatalf("Failed to get KnativeServing manifest: %v", err)
		}
		resources.AssertKnativeObsoleteResource(t, clients, names.Namespace,
			preManifest.Filter(mf.Not(mf.In(targetManifest))).Resources())
	})
}
