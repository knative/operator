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

package v1beta1

import (
	"reflect"
	"strings"
	"testing"

	"knative.dev/operator/pkg/apis/operator"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"knative.dev/operator/pkg/apis/operator/base"
	"knative.dev/pkg/apis"
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
	ks.MarkTargetClusterResolved()
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
	ks.MarkTargetClusterResolved()
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

func TestSecurityGuardDeprecationWarning(t *testing.T) {
	status := &KnativeServingStatus{}
	status.InitializeConditions()
	ready := *status.GetCondition(apis.ConditionReady)

	status.MarkSecurityGuardDeprecated()
	warning := status.GetCondition(base.SecurityGuardDeprecated)
	if warning == nil || !warning.IsTrue() || warning.Severity != apis.ConditionSeverityWarning || warning.Reason != "RemovalScheduled" {
		t.Fatalf("unexpected warning: %#v", warning)
	}
	if !strings.Contains(warning.Message, "Knative Operator v1.25") || !strings.Contains(warning.Message, "will have no effect") {
		t.Fatalf("warning must describe the upcoming removal: %q", warning.Message)
	}
	if got := status.GetCondition(apis.ConditionReady); !reflect.DeepEqual(*got, ready) {
		t.Fatal("warning changed pending readiness")
	}
	warned := status.DeepCopy()
	status.MarkSecurityGuardDeprecated()
	if !reflect.DeepEqual(status, warned) {
		t.Fatal("repeated warning changed status or transition timestamps")
	}

	status.MarkVersionMigrationEligible()
	status.MarkDependenciesInstalled()
	status.MarkInstallSucceeded()
	status.MarkDeploymentsAvailable()
	status.MarkTargetClusterResolved()
	if !status.IsReady() {
		t.Fatal("deprecation warning must not prevent readiness")
	}
	status.ClearSecurityGuardDeprecated()
	if status.GetCondition(base.SecurityGuardDeprecated) != nil || !status.IsReady() {
		t.Fatal("clearing the warning must preserve readiness")
	}
	cleared := status.DeepCopy()
	status.ClearSecurityGuardDeprecated()
	if !reflect.DeepEqual(status, cleared) {
		t.Fatal("clearing an absent warning changed status")
	}
}

func TestKnativeServingTargetClusterTransitions(t *testing.T) {
	t.Run("PreservesInstallSucceeded", func(t *testing.T) {
		ks := &KnativeServingStatus{}
		ks.InitializeConditions()

		// Drive the component to fully Ready first.
		ks.MarkVersionMigrationEligible()
		ks.MarkDependenciesInstalled()
		ks.MarkInstallSucceeded()
		ks.MarkDeploymentsAvailable()
		ks.MarkTargetClusterResolved()
		if ready := ks.IsReady(); !ready {
			t.Fatalf("precondition: ks.IsReady() = %v, want true", ready)
		}
		apistest.CheckConditionSucceeded(ks, base.InstallSucceeded, t)

		// Simulate a hub-side disconnect.
		ks.MarkTargetClusterNotResolved(base.ReasonClusterProfileNotReady, "control plane unhealthy")

		apistest.CheckConditionFailed(ks, base.TargetClusterResolved, t)
		tc := ks.GetCondition(base.TargetClusterResolved)
		if tc == nil || tc.Reason != base.ReasonClusterProfileNotReady {
			t.Fatalf("TargetClusterResolved.Reason = %v, want %q", tc, base.ReasonClusterProfileNotReady)
		}

		apistest.CheckConditionSucceeded(ks, base.InstallSucceeded, t)

		if ready := ks.IsReady(); ready {
			t.Fatalf("ks.IsReady() = %v, want false after MarkTargetClusterNotResolved", ready)
		}
	})

	t.Run("InitialDeployFailureIsUnknown", func(t *testing.T) {
		ks := &KnativeServingStatus{}
		ks.InitializeConditions()

		ks.MarkTargetClusterNotResolved(base.ReasonClusterProfileNotFound, "cluster profile not found")

		apistest.CheckConditionFailed(ks, base.TargetClusterResolved, t)
		apistest.CheckConditionOngoing(ks, base.InstallSucceeded, t)
		if ready := ks.IsReady(); ready {
			t.Fatalf("ks.IsReady() = %v, want false", ready)
		}
	})

	t.Run("ToggleDoesNotCorruptInstallSucceeded", func(t *testing.T) {
		ks := &KnativeServingStatus{}
		ks.InitializeConditions()

		ks.MarkVersionMigrationEligible()
		ks.MarkDependenciesInstalled()
		ks.MarkInstallSucceeded()
		ks.MarkDeploymentsAvailable()
		ks.MarkTargetClusterResolved()
		if ready := ks.IsReady(); !ready {
			t.Fatalf("precondition: ks.IsReady() = %v, want true", ready)
		}

		for i := range 3 {
			ks.MarkTargetClusterNotResolved(base.ReasonClusterProfileNotReady, "flapping")
			apistest.CheckConditionFailed(ks, base.TargetClusterResolved, t)
			apistest.CheckConditionSucceeded(ks, base.InstallSucceeded, t)
			if ready := ks.IsReady(); ready {
				t.Fatalf("iteration %d: ks.IsReady() = %v, want false", i, ready)
			}

			ks.MarkTargetClusterResolved()
			apistest.CheckConditionSucceeded(ks, base.TargetClusterResolved, t)
			apistest.CheckConditionSucceeded(ks, base.InstallSucceeded, t)
		}

		if ready := ks.IsReady(); !ready {
			t.Fatalf("final ks.IsReady() = %v, want true", ready)
		}
	})
}
