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
	"os"
	"testing"
	"time"

	mf "github.com/manifestival/manifestival"
	"knative.dev/operator/pkg/apis/operator/base"
	"knative.dev/operator/pkg/apis/operator/v1beta1"
	"knative.dev/operator/pkg/reconciler/common"
	util "knative.dev/operator/pkg/reconciler/common/testing"
	"knative.dev/operator/pkg/reconciler/knativeserving/ingress"
	"knative.dev/operator/test"
	"knative.dev/operator/test/client"
	"knative.dev/operator/test/resources"
	pkgupgrade "knative.dev/pkg/test/upgrade"
)

const (
	// TimeoutUpgrade specifies the timeout for Knative eventing upgrade.
	TimeoutUpgrade = 20 * time.Second
)

// OperatorPostUpgradeTests verifies the KnativeServing and KnativeEventing creation, deployment recreation, and
// the deletion after the operator upgrades to the latest release.
func OperatorPostUpgradeTests() []pkgupgrade.Operation {
	return []pkgupgrade.Operation{
		ServingCRPostUpgradeTests(),
		EventingCRPostUpgradeTests(),
	}
}

// ServingCRPostUpgradeTests verifies Knative Serving installation after the upgrade.
func ServingCRPostUpgradeTests() pkgupgrade.Operation {
	return pkgupgrade.NewOperation("ServingCRPostUpgradeTests", func(c pkgupgrade.Context) {
		servingCRPostUpgrade(c.T)
	})
}

// EventingCRPostUpgradeTests verifies Knative Eventing installation after the upgrade.
func EventingCRPostUpgradeTests() pkgupgrade.Operation {
	return pkgupgrade.NewOperation("EventingCRPostUpgradeTests", func(c pkgupgrade.Context) {
		eventingCRPostUpgrade(c.T)
	})
}

// EventingTimeoutForUpgrade adds a timeout for Knative Eventing to complete the upgrade for readiness.
func EventingTimeoutForUpgrade() pkgupgrade.Operation {
	return pkgupgrade.NewOperation("EventingTimeoutForUpgrade", func(c pkgupgrade.Context) {
		// Operator has the issue making sure all deployments are ready before running the postupgrade
		// tests, especially when spec.version is set to latest. Before figuring out the optimal approach,
		// we add a timeout of 20 seconds here to make sure all the deployments are up and running for the
		// target version.
		time.Sleep(TimeoutUpgrade)
	})
}

func servingCRPostUpgrade(t *testing.T) {
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
		ks := &v1beta1.KnativeServing{
			Spec: v1beta1.KnativeServingSpec{
				CommonSpec: base.CommonSpec{
					Version: common.LATEST_VERSION,
				},
			},
		}
		targetManifest, err := common.TargetManifest(ks)
		if err != nil {
			t.Fatalf("Failed to get the manifest for Knative: %v", err)
		}
		expectedDeployments := resources.GetExpectedDeployments(targetManifest.Filter(ingress.Filters(ks)))
		util.AssertEqual(t, len(expectedDeployments) > 0, true)
		resources.AssertKnativeDeploymentStatus(t, clients, names.Namespace, common.TargetVersion(ks), test.OperatorFlags.PreviousServingVersion,
			expectedDeployments)
		resources.AssertKSOperatorCRReadyStatus(t, clients, names)

		instance := &v1beta1.KnativeServing{
			Spec: v1beta1.KnativeServingSpec{
				CommonSpec: base.CommonSpec{
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
		targetManifest, _ = targetManifest.Transform(mf.InjectNamespace(names.Namespace))
		preManifest, _ = preManifest.Transform(mf.InjectNamespace(names.Namespace))
		resources.AssertKnativeObsoleteResource(t, clients, names.Namespace,
			preManifest.Filter(mf.Not(mf.In(targetManifest))).Resources())
	})
}

func eventingCRPostUpgrade(t *testing.T) {
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
		resources.SetKodataDir()
		defer os.Unsetenv(common.KoEnvKey)
		// Based on the latest release version, get the deployment resources.
		ke := &v1beta1.KnativeEventing{
			Spec: v1beta1.KnativeEventingSpec{
				CommonSpec: base.CommonSpec{
					Version: common.LATEST_VERSION,
				},
			},
		}
		targetManifest, err := common.TargetManifest(ke)
		if err != nil {
			t.Fatalf("Failed to get the manifest for Knative: %v", err)
		}
		expectedDeployments := resources.GetExpectedDeployments(targetManifest)
		util.AssertEqual(t, len(expectedDeployments) > 0, true)
		resources.AssertKnativeDeploymentStatus(t, clients, names.Namespace, common.TargetVersion(ke), test.OperatorFlags.PreviousEventingVersion,
			expectedDeployments)

		instance := &v1beta1.KnativeEventing{
			Spec: v1beta1.KnativeEventingSpec{
				CommonSpec: base.CommonSpec{
					Version: test.OperatorFlags.PreviousEventingVersion,
				},
			},
		}
		// Compare the previous manifest with the target manifest, we verify that all the obsolete resources
		// do not exist any more.
		preManifest, err := common.TargetManifest(instance)
		if err != nil {
			t.Fatalf("Failed to get KnativeEventing manifest: %v", err)
		}
		targetManifest, _ = targetManifest.Transform(mf.InjectNamespace(names.Namespace))
		preManifest, _ = preManifest.Transform(mf.InjectNamespace(names.Namespace))
		resources.AssertKnativeObsoleteResource(t, clients, names.Namespace,
			preManifest.Filter(mf.Not(mf.In(targetManifest))).Resources())
	})
}
