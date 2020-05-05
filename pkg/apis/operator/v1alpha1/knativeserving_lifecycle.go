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
	"k8s.io/apimachinery/pkg/runtime/schema"
	"knative.dev/pkg/apis"
)

var conditions = apis.NewLivingConditionSet(
	DependenciesInstalled,
	DeploymentsAvailable,
	InstallSucceeded,
)

// GroupVersionKind returns SchemeGroupVersion of a KnativeServing
func (ks *KnativeServing) GroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind(KindKnativeServing)
}

// GetCondition returns the current condition of a given condition type
func (is *KnativeServingStatus) GetCondition(t apis.ConditionType) *apis.Condition {
	return conditions.Manage(is).GetCondition(t)
}

// InitializeConditions initializes conditions of an KnativeServingStatus
func (is *KnativeServingStatus) InitializeConditions() {
	conditions.Manage(is).InitializeConditions()
}

// IsReady looks at the conditions returns true if they are all true.
func (is *KnativeServingStatus) IsReady() bool {
	return conditions.Manage(is).IsHappy()
}

// MarkInstallSucceeded marks the InstallationSucceeded status as true.
func (is *KnativeServingStatus) MarkInstallSucceeded() {
	conditions.Manage(is).MarkTrue(InstallSucceeded)
	if is.GetCondition(DependenciesInstalled).IsUnknown() {
		// Assume deps are installed if we're not sure
		is.MarkDependenciesInstalled()
	}
}

// MarkInstallFailed marks the InstallationSucceeded status as false with the given
// message.
func (is *KnativeServingStatus) MarkInstallFailed(msg string) {
	conditions.Manage(is).MarkFalse(
		InstallSucceeded,
		"Error",
		"Install failed with message: %s", msg)
}

// MarkDeploymentsAvailable marks the DeploymentsAvailable status as true.
func (is *KnativeServingStatus) MarkDeploymentsAvailable() {
	conditions.Manage(is).MarkTrue(DeploymentsAvailable)
}

// MarkDeploymentsNotReady marks the DeploymentsAvailable status as false and calls out
// it's waiting for deployments.
func (is *KnativeServingStatus) MarkDeploymentsNotReady() {
	conditions.Manage(is).MarkFalse(
		DeploymentsAvailable,
		"NotReady",
		"Waiting on deployments")
}

// MarkDependenciesInstalled marks the DependenciesInstalled status as true.
func (is *KnativeServingStatus) MarkDependenciesInstalled() {
	conditions.Manage(is).MarkTrue(DependenciesInstalled)
}

// MarkDependencyInstalling marks the DependenciesInstalled status as false with the
// given message.
func (is *KnativeServingStatus) MarkDependencyInstalling(msg string) {
	conditions.Manage(is).MarkFalse(
		DependenciesInstalled,
		"Installing",
		"Dependency installing: %s", msg)
}

// MarkDependencyMissing marks the DependenciesInstalled status as false with the
// given message.
func (is *KnativeServingStatus) MarkDependencyMissing(msg string) {
	conditions.Manage(is).MarkFalse(
		DependenciesInstalled,
		"Error",
		"Dependency missing: %s", msg)
}
