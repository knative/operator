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
	return SchemeGroupVersion.WithKind(Kind)
}

// GetConditions implements apis.ConditionsAccessor
func (is *KnativeServingStatus) GetConditions() apis.Conditions {
	return is.Conditions
}

// SetConditions implements apis.ConditionsAccessor
func (is *KnativeServingStatus) SetConditions(c apis.Conditions) {
	is.Conditions = c
}

func (is *KnativeServingStatus) IsReady() bool {
	return conditions.Manage(is).IsHappy()
}

func (is *KnativeServingStatus) IsInstalled() bool {
	return is.GetCondition(InstallSucceeded).IsTrue()
}

func (is *KnativeServingStatus) IsAvailable() bool {
	return is.GetCondition(DeploymentsAvailable).IsTrue()
}

func (is *KnativeServingStatus) IsDeploying() bool {
	return is.IsInstalled() && !is.IsAvailable()
}

func (is *KnativeServingStatus) IsFullySupported() bool {
	return is.GetCondition(DependenciesInstalled).IsTrue()
}

func (is *KnativeServingStatus) GetCondition(t apis.ConditionType) *apis.Condition {
	return conditions.Manage(is).GetCondition(t)
}

func (is *KnativeServingStatus) InitializeConditions() {
	conditions.Manage(is).InitializeConditions()
}

func (is *KnativeServingStatus) MarkInstallFailed(msg string) {
	conditions.Manage(is).MarkFalse(
		InstallSucceeded,
		"Error",
		"Install failed with message: %s", msg)
}

func (is *KnativeServingStatus) MarkInstallSucceeded() {
	conditions.Manage(is).MarkTrue(InstallSucceeded)
	if is.GetCondition(DependenciesInstalled).IsUnknown() {
		// Assume deps are installed if we're not sure
		is.MarkDependenciesInstalled()
	}
}

func (is *KnativeServingStatus) MarkDeploymentsAvailable() {
	conditions.Manage(is).MarkTrue(DeploymentsAvailable)
}

func (is *KnativeServingStatus) MarkDeploymentsNotReady() {
	conditions.Manage(is).MarkFalse(
		DeploymentsAvailable,
		"NotReady",
		"Waiting on deployments")
}

func (is *KnativeServingStatus) MarkDependenciesInstalled() {
	conditions.Manage(is).MarkTrue(DependenciesInstalled)
}

func (is *KnativeServingStatus) MarkDependencyInstalling(msg string) {
	conditions.Manage(is).MarkFalse(
		DependenciesInstalled,
		"Installing",
		"Dependency installing: %s", msg)
}

func (is *KnativeServingStatus) MarkDependencyMissing(msg string) {
	conditions.Manage(is).MarkFalse(
		DependenciesInstalled,
		"Error",
		"Dependency missing: %s", msg)
}
