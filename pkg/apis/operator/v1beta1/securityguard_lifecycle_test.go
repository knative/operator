/*
Copyright 2026 The Knative Authors

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
	"testing"

	"knative.dev/operator/pkg/apis/operator/base"
	"knative.dev/pkg/apis"
)

func TestSecurityGuardWarningLifecycle(t *testing.T) {
	for _, ready := range []bool{false, true} {
		t.Run(map[bool]string{false: "pending", true: "ready"}[ready], func(t *testing.T) {
			status := &KnativeServingStatus{}
			status.InitializeConditions()
			if ready {
				status.MarkVersionMigrationEligible()
				status.MarkDependenciesInstalled()
				status.MarkInstallSucceeded()
				status.MarkDeploymentsAvailable()
				status.MarkTargetClusterResolved()
			}
			before := *status.GetCondition(apis.ConditionReady)
			status.MarkSecurityGuardDeprecated()
			cond := status.GetCondition(base.SecurityGuardDeprecated)
			if cond == nil || !cond.IsTrue() || cond.Severity != apis.ConditionSeverityWarning || cond.Reason != "RemovalScheduled" {
				t.Fatalf("unexpected warning: %#v", cond)
			}
			if cond.Message != "Security Guard is deprecated and will no longer be supported in Knative Operator v1.25. spec.security.securityGuard.enabled will have no effect after upgrading the Operator." {
				t.Fatalf("unexpected warning message: %q", cond.Message)
			}
			warned := status.DeepCopy()
			status.MarkSecurityGuardDeprecated()
			if !reflect.DeepEqual(status, warned) {
				t.Fatal("repeated warnings churn status or transition timestamps")
			}
			if got := status.GetCondition(apis.ConditionReady); !reflect.DeepEqual(*got, before) || status.IsReady() != ready {
				t.Fatal("warning changed readiness")
			}
			status.ClearSecurityGuardDeprecated()
			if status.GetCondition(base.SecurityGuardDeprecated) != nil {
				t.Fatal("warning was not removed")
			}
			cleared := status.DeepCopy()
			status.ClearSecurityGuardDeprecated()
			if !reflect.DeepEqual(status, cleared) {
				t.Fatal("clearing an absent warning changed status")
			}
			if got := status.GetCondition(apis.ConditionReady); !reflect.DeepEqual(*got, before) || status.IsReady() != ready {
				t.Fatal("clearing the warning changed readiness")
			}
		})
	}
}
