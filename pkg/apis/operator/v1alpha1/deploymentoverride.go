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

package v1alpha1

import (
	"knative.dev/operator/pkg/apis/operator/base"
)

// ConvertToDeploymentOverride merges the ResourceRequirementsOverride into the DeploymentOverride
func ConvertToDeploymentOverride(source base.KComponent) []base.DeploymentOverride {
	mergedDeploymentOverride := source.GetSpec().GetDeploymentOverride()
	// Make a copy of source.GetSpec().GetDeploymentOverride()
	deploymentOverrideCopy := make([]base.DeploymentOverride, 0, len(mergedDeploymentOverride))
	for _, override := range mergedDeploymentOverride {
		copy := *override.DeepCopy()
		deploymentOverrideCopy = append(deploymentOverrideCopy, copy)
	}

	for _, resource := range source.GetSpec().GetResources() {
		resourceCopy := resource.DeepCopy()
		deploymentOverrideCopy = addResourceIntoDeployment(deploymentOverrideCopy, *resourceCopy)
	}
	return deploymentOverrideCopy
}

func addResourceIntoDeployment(deploymentOverrides []base.DeploymentOverride,
	resource base.ResourceRequirementsOverride) []base.DeploymentOverride {
	// If it does not exist, add the resource requirement as a new
	// item; if it does, modify the existing resource requirement.
	deployFound := false
	for key, deploymentOverride := range deploymentOverrides {
		if deploymentOverride.Name == resource.Container {
			deployFound = true
			containerFound := false
			for containerKey, deployResource := range deploymentOverride.Resources {
				if deployResource.Container == resource.Container {
					containerFound = true
					if len(deployResource.Limits) == 0 && len(deployResource.Requests) == 0 {
						deploymentOverrides[key].Resources[containerKey].Limits = resource.Limits
						deploymentOverrides[key].Resources[containerKey].Requests = resource.Requests
					}
				}
			}

			if !containerFound {
				deploymentOverrides[key].Resources = append(deploymentOverrides[key].Resources, resource)
			}
		}
	}
	if !deployFound {
		newDeployOverride := base.DeploymentOverride{}
		// Take the container name as the deployment name.
		newDeployOverride.Name = resource.Container
		newDeployOverride.Resources = append(newDeployOverride.Resources, resource)
		deploymentOverrides = append(deploymentOverrides, newDeployOverride)
	}
	return deploymentOverrides
}
