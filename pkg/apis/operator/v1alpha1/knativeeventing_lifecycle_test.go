/*
Copyright 2019 The Knative Authors

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

package v1alpha1

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"knative.dev/operator/pkg/apis/operator/base"
	apistest "knative.dev/pkg/apis/testing"
)

func TestKnativeEventingGroupVersionKind(t *testing.T) {
	r := &KnativeEventing{}
	want := schema.GroupVersionKind{
		Group:   GroupName,
		Version: SchemaVersion,
		Kind:    KindKnativeEventing,
	}
	if got := r.GroupVersionKind(); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

func TestKnativeEventingHappyPath(t *testing.T) {
	ke := &KnativeEventingStatus{}
	ke.InitializeConditions()

	apistest.CheckConditionOngoing(ke, base.DependenciesInstalled, t)
	apistest.CheckConditionOngoing(ke, base.DeploymentsAvailable, t)
	apistest.CheckConditionOngoing(ke, base.InstallSucceeded, t)

	ke.MarkVersionMigrationEligible()

	// Install succeeds.
	ke.MarkInstallSucceeded()
	// Dependencies are assumed successful too.
	apistest.CheckConditionSucceeded(ke, base.DependenciesInstalled, t)
	apistest.CheckConditionOngoing(ke, base.DeploymentsAvailable, t)
	apistest.CheckConditionSucceeded(ke, base.InstallSucceeded, t)

	// Deployments are not available at first.
	ke.MarkDeploymentsNotReady([]string{"test"})
	apistest.CheckConditionSucceeded(ke, base.DependenciesInstalled, t)
	apistest.CheckConditionFailed(ke, base.DeploymentsAvailable, t)
	apistest.CheckConditionSucceeded(ke, base.InstallSucceeded, t)
	if ready := ke.IsReady(); ready {
		t.Errorf("ke.IsReady() = %v, want false", ready)
	}

	// Deployments become ready and we're good.
	ke.MarkDeploymentsAvailable()
	apistest.CheckConditionSucceeded(ke, base.DependenciesInstalled, t)
	apistest.CheckConditionSucceeded(ke, base.DeploymentsAvailable, t)
	apistest.CheckConditionSucceeded(ke, base.InstallSucceeded, t)
	if ready := ke.IsReady(); !ready {
		t.Errorf("ke.IsReady() = %v, want true", ready)
	}
}

func TestKnativeEventingErrorPath(t *testing.T) {
	ke := &KnativeEventingStatus{}
	ke.InitializeConditions()

	apistest.CheckConditionOngoing(ke, base.DependenciesInstalled, t)
	apistest.CheckConditionOngoing(ke, base.DeploymentsAvailable, t)
	apistest.CheckConditionOngoing(ke, base.InstallSucceeded, t)

	ke.MarkVersionMigrationEligible()

	// Install fails.
	ke.MarkInstallFailed("test")
	apistest.CheckConditionOngoing(ke, base.DependenciesInstalled, t)
	apistest.CheckConditionOngoing(ke, base.DeploymentsAvailable, t)
	apistest.CheckConditionFailed(ke, base.InstallSucceeded, t)

	// Dependencies are installing.
	ke.MarkDependencyInstalling("testing")
	apistest.CheckConditionFailed(ke, base.DependenciesInstalled, t)
	apistest.CheckConditionOngoing(ke, base.DeploymentsAvailable, t)
	apistest.CheckConditionFailed(ke, base.InstallSucceeded, t)

	// Install now succeeds.
	ke.MarkInstallSucceeded()
	apistest.CheckConditionFailed(ke, base.DependenciesInstalled, t)
	apistest.CheckConditionOngoing(ke, base.DeploymentsAvailable, t)
	apistest.CheckConditionSucceeded(ke, base.InstallSucceeded, t)
	if ready := ke.IsReady(); ready {
		t.Errorf("ke.IsReady() = %v, want false", ready)
	}

	// Deployments become ready
	ke.MarkDeploymentsAvailable()
	apistest.CheckConditionFailed(ke, base.DependenciesInstalled, t)
	apistest.CheckConditionSucceeded(ke, base.DeploymentsAvailable, t)
	apistest.CheckConditionSucceeded(ke, base.InstallSucceeded, t)
	if ready := ke.IsReady(); ready {
		t.Errorf("ke.IsReady() = %v, want false", ready)
	}

	// Finally, dependencies become available.
	ke.MarkDependenciesInstalled()
	apistest.CheckConditionSucceeded(ke, base.DependenciesInstalled, t)
	apistest.CheckConditionSucceeded(ke, base.DeploymentsAvailable, t)
	apistest.CheckConditionSucceeded(ke, base.InstallSucceeded, t)
	if ready := ke.IsReady(); !ready {
		t.Errorf("ke.IsReady() = %v, want true", ready)
	}
}

func TestKnativeEventingExternalDependency(t *testing.T) {
	ke := &KnativeEventingStatus{}
	ke.InitializeConditions()

	// External marks dependency as failed.
	ke.MarkDependencyMissing("test")

	// Install succeeds.
	ke.MarkInstallSucceeded()
	apistest.CheckConditionFailed(ke, base.DependenciesInstalled, t)
	apistest.CheckConditionOngoing(ke, base.DeploymentsAvailable, t)
	apistest.CheckConditionSucceeded(ke, base.InstallSucceeded, t)

	// Dependencies are now ready.
	ke.MarkDependenciesInstalled()
	apistest.CheckConditionSucceeded(ke, base.DependenciesInstalled, t)
	apistest.CheckConditionOngoing(ke, base.DeploymentsAvailable, t)
	apistest.CheckConditionSucceeded(ke, base.InstallSucceeded, t)
}

func TestKnativeEventingVersionMigrationNotEligible(t *testing.T) {
	ke := &KnativeEventingStatus{}
	ke.InitializeConditions()

	ke.MarkVersionMigrationNotEligible("Version migration not eligible.")
	apistest.CheckConditionFailed(ke, base.VersionMigrationEligible, t)
}
