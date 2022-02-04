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

	"knative.dev/operator/pkg/apis/operator"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"knative.dev/operator/pkg/apis/operator/base"
	apistest "knative.dev/pkg/apis/testing"
)

func TestKnativeServingGroupVersionKind(t *testing.T) {
	r := &KnativeServing{}
	want := schema.GroupVersionKind{
		Group:   operator.GroupName,
		Version: SchemaVersion,
		Kind:    operator.KindKnativeServing,
	}
	if got := r.GroupVersionKind(); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

func TestKnativeServingHappyPath(t *testing.T) {
	ks := &KnativeServingStatus{}
	ks.InitializeConditions()

	apistest.CheckConditionOngoing(ks, base.DependenciesInstalled, t)
	apistest.CheckConditionOngoing(ks, base.DeploymentsAvailable, t)
	apistest.CheckConditionOngoing(ks, base.InstallSucceeded, t)

	ks.MarkVersionMigrationEligible()

	// Install succeeds.
	ks.MarkInstallSucceeded()
	// Dependencies are assumed successful too.
	apistest.CheckConditionSucceeded(ks, base.DependenciesInstalled, t)
	apistest.CheckConditionOngoing(ks, base.DeploymentsAvailable, t)
	apistest.CheckConditionSucceeded(ks, base.InstallSucceeded, t)

	// Deployments are not available at first.
	ks.MarkDeploymentsNotReady([]string{"test"})
	apistest.CheckConditionSucceeded(ks, base.DependenciesInstalled, t)
	apistest.CheckConditionFailed(ks, base.DeploymentsAvailable, t)
	apistest.CheckConditionSucceeded(ks, base.InstallSucceeded, t)
	if ready := ks.IsReady(); ready {
		t.Errorf("ks.IsReady() = %v, want false", ready)
	}

	// Deployments become ready and we're good.
	ks.MarkDeploymentsAvailable()
	apistest.CheckConditionSucceeded(ks, base.DependenciesInstalled, t)
	apistest.CheckConditionSucceeded(ks, base.DeploymentsAvailable, t)
	apistest.CheckConditionSucceeded(ks, base.InstallSucceeded, t)
	if ready := ks.IsReady(); !ready {
		t.Errorf("ks.IsReady() = %v, want true", ready)
	}
}

func TestKnativeServingErrorPath(t *testing.T) {
	ks := &KnativeServingStatus{}
	ks.InitializeConditions()

	apistest.CheckConditionOngoing(ks, base.DependenciesInstalled, t)
	apistest.CheckConditionOngoing(ks, base.DeploymentsAvailable, t)
	apistest.CheckConditionOngoing(ks, base.InstallSucceeded, t)

	ks.MarkVersionMigrationEligible()

	// Install fails.
	ks.MarkInstallFailed("test")
	apistest.CheckConditionOngoing(ks, base.DependenciesInstalled, t)
	apistest.CheckConditionOngoing(ks, base.DeploymentsAvailable, t)
	apistest.CheckConditionFailed(ks, base.InstallSucceeded, t)

	// Dependencies are installing.
	ks.MarkDependencyInstalling("testing")
	apistest.CheckConditionFailed(ks, base.DependenciesInstalled, t)
	apistest.CheckConditionOngoing(ks, base.DeploymentsAvailable, t)
	apistest.CheckConditionFailed(ks, base.InstallSucceeded, t)

	// Install now succeeds.
	ks.MarkInstallSucceeded()
	apistest.CheckConditionFailed(ks, base.DependenciesInstalled, t)
	apistest.CheckConditionOngoing(ks, base.DeploymentsAvailable, t)
	apistest.CheckConditionSucceeded(ks, base.InstallSucceeded, t)
	if ready := ks.IsReady(); ready {
		t.Errorf("ks.IsReady() = %v, want false", ready)
	}

	// Deployments become ready
	ks.MarkDeploymentsAvailable()
	apistest.CheckConditionFailed(ks, base.DependenciesInstalled, t)
	apistest.CheckConditionSucceeded(ks, base.DeploymentsAvailable, t)
	apistest.CheckConditionSucceeded(ks, base.InstallSucceeded, t)
	if ready := ks.IsReady(); ready {
		t.Errorf("ks.IsReady() = %v, want false", ready)
	}

	// Finally, dependencies become available.
	ks.MarkDependenciesInstalled()
	apistest.CheckConditionSucceeded(ks, base.DependenciesInstalled, t)
	apistest.CheckConditionSucceeded(ks, base.DeploymentsAvailable, t)
	apistest.CheckConditionSucceeded(ks, base.InstallSucceeded, t)
	if ready := ks.IsReady(); !ready {
		t.Errorf("ks.IsReady() = %v, want true", ready)
	}
}

func TestKnativeServingExternalDependency(t *testing.T) {
	ks := &KnativeServingStatus{}
	ks.InitializeConditions()

	// External marks dependency as failed.
	ks.MarkDependencyMissing("test")

	// Install succeeds.
	ks.MarkInstallSucceeded()
	apistest.CheckConditionFailed(ks, base.DependenciesInstalled, t)
	apistest.CheckConditionOngoing(ks, base.DeploymentsAvailable, t)
	apistest.CheckConditionSucceeded(ks, base.InstallSucceeded, t)

	// Dependencies are now ready.
	ks.MarkDependenciesInstalled()
	apistest.CheckConditionSucceeded(ks, base.DependenciesInstalled, t)
	apistest.CheckConditionOngoing(ks, base.DeploymentsAvailable, t)
	apistest.CheckConditionSucceeded(ks, base.InstallSucceeded, t)
}

func TestKnativeServingVersionMigrationNotEligible(t *testing.T) {
	ks := &KnativeServingStatus{}
	ks.InitializeConditions()

	ks.MarkVersionMigrationNotEligible("Version migration not eligible.")
	apistest.CheckConditionFailed(ks, base.VersionMigrationEligible, t)
}
