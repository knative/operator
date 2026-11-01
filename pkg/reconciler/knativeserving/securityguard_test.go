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

package knativeserving

import (
	"context"
	"testing"

	"knative.dev/operator/pkg/apis/operator/base"
	"knative.dev/operator/pkg/apis/operator/v1beta1"
)

func TestSecurityGuardDeprecationWarningClearsBeforeMigrationError(t *testing.T) {
	instance := &v1beta1.KnativeServing{
		Spec: v1beta1.KnativeServingSpec{
			CommonSpec: base.CommonSpec{Version: "1.24.0"},
			Security:   &v1beta1.SecurityConfigs{SecurityGuard: base.SecurityGuardConfiguration{Enabled: true}},
		},
		Status: v1beta1.KnativeServingStatus{Version: "2.0.0"},
	}
	// A migration error exercises warning updates before any later stage runs.
	reconciler := &Reconciler{}
	for _, security := range []*v1beta1.SecurityConfigs{
		instance.Spec.Security,
		{SecurityGuard: base.SecurityGuardConfiguration{Enabled: false}},
		nil,
	} {
		instance.Spec.Security = security
		instance.Status.MarkSecurityGuardDeprecated()
		if err := reconciler.ReconcileKind(context.Background(), instance); err != nil {
			t.Fatal(err)
		}
		cond := instance.Status.GetCondition(base.SecurityGuardDeprecated)
		if security != nil && security.SecurityGuard.Enabled {
			if cond == nil || !cond.IsTrue() {
				t.Fatal("enabled field must warn about its removal")
			}
		} else if cond != nil {
			t.Fatal("false or absent configuration must clear the warning")
		}
		if !instance.Status.GetCondition(base.VersionMigrationEligible).IsFalse() {
			t.Fatal("expected migration error")
		}
	}
}
