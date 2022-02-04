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
	"strings"

	"knative.dev/operator/pkg/apis/operator"
	"knative.dev/operator/pkg/apis/operator/base"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"knative.dev/pkg/apis"
)

var (
	_ base.KComponentStatus = (*KnativeServingStatus)(nil)

	servingCondSet = apis.NewLivingConditionSet(
		base.DependenciesInstalled,
		base.DeploymentsAvailable,
		base.InstallSucceeded,
		base.VersionMigrationEligible,
	)
)

// GroupVersionKind returns SchemeGroupVersion of a KnativeServing
func (ks *KnativeServing) GroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind(operator.KindKnativeServing)
}

// GetCondition returns the current condition of a given condition type
func (is *KnativeServingStatus) GetCondition(t apis.ConditionType) *apis.Condition {
	return servingCondSet.Manage(is).GetCondition(t)
}

// InitializeConditions initializes conditions of an KnativeServingStatus
func (is *KnativeServingStatus) InitializeConditions() {
	servingCondSet.Manage(is).InitializeConditions()
}

// IsReady looks at the conditions returns true if they are all true.
func (is *KnativeServingStatus) IsReady() bool {
	return servingCondSet.Manage(is).IsHappy()
}

// MarkInstallSucceeded marks the InstallationSucceeded status as true.
func (is *KnativeServingStatus) MarkInstallSucceeded() {
	servingCondSet.Manage(is).MarkTrue(base.InstallSucceeded)
	if is.GetCondition(base.DependenciesInstalled).IsUnknown() {
		// Assume deps are installed if we're not sure
		is.MarkDependenciesInstalled()
	}
}

// MarkInstallFailed marks the InstallationSucceeded status as false with the given
// message.
func (is *KnativeServingStatus) MarkInstallFailed(msg string) {
	servingCondSet.Manage(is).MarkFalse(
		base.InstallSucceeded,
		"Error",
		"Install failed with message: %s", msg)
}

// MarkVersionMigrationEligible marks the VersionMigrationEligible status as false with given message.
func (is *KnativeServingStatus) MarkVersionMigrationEligible() {
	servingCondSet.Manage(is).MarkTrue(base.VersionMigrationEligible)
}

// MarkVersionMigrationNotEligible marks the DeploymentsAvailable status as true.
func (is *KnativeServingStatus) MarkVersionMigrationNotEligible(msg string) {
	servingCondSet.Manage(is).MarkFalse(
		base.VersionMigrationEligible,
		"Error",
		"Version migration is not eligible with message: %s", msg)
}

// MarkDeploymentsAvailable marks the DeploymentsAvailable status as true.
func (is *KnativeServingStatus) MarkDeploymentsAvailable() {
	servingCondSet.Manage(is).MarkTrue(base.DeploymentsAvailable)
}

// MarkDeploymentsNotReady marks the DeploymentsAvailable status as false and calls out
// it's waiting for deployments.
func (is *KnativeServingStatus) MarkDeploymentsNotReady(deployments []string) {
	servingCondSet.Manage(is).MarkFalse(
		base.DeploymentsAvailable,
		"NotReady",
		"Waiting on deployments: %s", strings.Join(deployments, ", "))
}

// MarkDependenciesInstalled marks the DependenciesInstalled status as true.
func (is *KnativeServingStatus) MarkDependenciesInstalled() {
	servingCondSet.Manage(is).MarkTrue(base.DependenciesInstalled)
}

// MarkDependencyInstalling marks the DependenciesInstalled status as false with the
// given message.
func (is *KnativeServingStatus) MarkDependencyInstalling(msg string) {
	servingCondSet.Manage(is).MarkFalse(
		base.DependenciesInstalled,
		"Installing",
		"Dependency installing: %s", msg)
}

// MarkDependencyMissing marks the DependenciesInstalled status as false with the
// given message.
func (is *KnativeServingStatus) MarkDependencyMissing(msg string) {
	servingCondSet.Manage(is).MarkFalse(
		base.DependenciesInstalled,
		"Error",
		"Dependency missing: %s", msg)
}

// GetVersion gets the currently installed version of the component.
func (is *KnativeServingStatus) GetVersion() string {
	return is.Version
}

// SetVersion sets the currently installed version of the component.
func (is *KnativeServingStatus) SetVersion(version string) {
	is.Version = version
}

// GetManifests gets the url links of the manifests.
func (is *KnativeServingStatus) GetManifests() []string {
	return is.Manifests
}

// SetManifests sets the url links of the manifests.
func (is *KnativeServingStatus) SetManifests(manifests []string) {
	is.Manifests = manifests
}
