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

package upgrade

import (
	"context"
	"os"
	"testing"

	"knative.dev/operator/pkg/apis/operator/base"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
	"knative.dev/operator/pkg/reconciler/common"
	util "knative.dev/operator/pkg/reconciler/common/testing"
	"knative.dev/operator/test"
	"knative.dev/operator/test/client"
	"knative.dev/operator/test/resources"
	pkgupgrade "knative.dev/pkg/test/upgrade"
)

// OperatorPreUpgradeTests verifies the KnativeServing and KnativeEventing creation, before upgraded to the latest HEAD.
func OperatorPreUpgradeTests() []pkgupgrade.Operation {
	return []pkgupgrade.Operation{
		ServingCRPreUpgradeTests(),
		EventingCRPreUpgradeTests(),
	}
}

// ServingCRPreUpgradeTests verifies the KnativeServing creation for the previous release.
func ServingCRPreUpgradeTests() pkgupgrade.Operation {
	return pkgupgrade.NewOperation("ServingCRPreUpgradeTests", func(c pkgupgrade.Context) {
		servingCRPreUpgrade(c.T)
	})
}

// EventingCRPreUpgradeTests verifies the KnativeEventing creation for the previous release.
func EventingCRPreUpgradeTests() pkgupgrade.Operation {
	return pkgupgrade.NewOperation("EventingCRPreUpgradeTests", func(c pkgupgrade.Context) {
		eventingCRPreUpgrade(c.T)
	})
}

func servingCRPreUpgrade(t *testing.T) {
	clients := client.Setup(t)
	names := test.ResourceNames{
		KnativeServing: test.OperatorName,
		Namespace:      test.ServingOperatorNamespace,
	}

	// Create a KnativeServing
	if _, err := resources.EnsureKnativeServingExists(clients.KnativeServing(), names); err != nil {
		t.Fatalf("KnativeServing %q failed to create: %v", names.KnativeServing, err)
	}

	// Verify if resources match the requirement for the previous release before upgrade
	t.Run("verify resources", func(t *testing.T) {
		resources.AssertKSOperatorCRReadyStatus(t, clients, names)
		kserving, err := clients.KnativeServing().Get(context.TODO(), names.KnativeServing, metav1.GetOptions{})
		if err != nil {
			t.Fatalf("Failed to get KnativeServing CR: %v", err)
		}
		resources.SetKodataDir()
		// Based on the status.version, get the deployment resources.
		defer os.Unsetenv(common.KoEnvKey)
		// The operator and the test cases do not share the same path of the kodata, we need to retrieve the
		// installed manifests in terms of the spec.
		kservingInstalled := &v1alpha1.KnativeServing{
			Spec: v1alpha1.KnativeServingSpec{
				CommonSpec: base.CommonSpec{
					Version: kserving.GetStatus().GetVersion(),
				},
			},
		}
		manifest, err := common.TargetManifest(kservingInstalled)
		if err != nil {
			t.Fatalf("Failed to get the manifest for Knative: %v", err)
		}
		expectedDeployments := resources.GetExpectedDeployments(manifest)
		util.AssertEqual(t, len(expectedDeployments) > 0, true)
		resources.AssertKnativeDeploymentStatus(t, clients, names.Namespace, kserving.GetStatus().GetVersion(), "",
			expectedDeployments)
	})
}

func eventingCRPreUpgrade(t *testing.T) {
	clients := client.Setup(t)
	names := test.ResourceNames{
		KnativeEventing: test.OperatorName,
		Namespace:       test.EventingOperatorNamespace,
	}

	// Create a KnativeEventing
	if _, err := resources.EnsureKnativeEventingExists(clients.KnativeEventing(), names); err != nil {
		t.Fatalf("KnativeEventing %q failed to create: %v", names.KnativeEventing, err)
	}

	// Verify if resources match the requirement for the previous release before upgrade
	t.Run("verify resources", func(t *testing.T) {
		resources.AssertKEOperatorCRReadyStatus(t, clients, names)
		keventing, err := clients.KnativeEventing().Get(context.TODO(), names.KnativeEventing, metav1.GetOptions{})
		if err != nil {
			t.Fatalf("Failed to get KnativeEventing CR: %v", err)
		}
		resources.SetKodataDir()
		// Based on the status.version, get the deployment resources.
		defer os.Unsetenv(common.KoEnvKey)
		// The operator and the test cases do not share the same path of the kodata, we need to retrieve the
		// installed manifests in terms of the spec.
		keventingInstalled := &v1alpha1.KnativeEventing{
			Spec: v1alpha1.KnativeEventingSpec{
				CommonSpec: base.CommonSpec{
					Version: keventing.GetStatus().GetVersion(),
				},
			},
		}
		manifest, err := common.TargetManifest(keventingInstalled)
		if err != nil {
			t.Fatalf("Failed to get the manifest for Knative: %v", err)
		}
		expectedDeployments := resources.GetExpectedDeployments(manifest)
		util.AssertEqual(t, len(expectedDeployments) > 0, true)
		resources.AssertKnativeDeploymentStatus(t, clients, names.Namespace, keventing.GetStatus().GetVersion(), "",
			expectedDeployments)
	})
}
