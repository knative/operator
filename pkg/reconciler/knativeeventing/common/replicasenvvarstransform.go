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

package common

import (
	mf "github.com/manifestival/manifestival"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes/scheme"
	"knative.dev/pkg/system"
)

var (
	envVarNames = sets.NewString(system.NamespaceEnvKey, "K_METRICS_CONFIG", "K_LOGGING_CONFIG",
		"K_LEADER_ELECTION_CONFIG", "K_NO_SHUTDOWN_AFTER", "K_SINK_TIMEOUT")
)

type unstructuredGetter interface {
	Get(obj *unstructured.Unstructured) (*unstructured.Unstructured, error)
}

// ReplicasEnvVarsTransform keeps the number of replicas and the env vars, if the deployment
// pingsource-mt-adapter exists in the cluster.
func ReplicasEnvVarsTransform(client unstructuredGetter) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if u.GetKind() == "Deployment" && u.GetName() == "pingsource-mt-adapter" {
			currentU, err := client.Get(u)
			if errors.IsNotFound(err) {
				return nil
			}
			if err != nil {
				return err
			}
			apply := &appsv1.Deployment{}
			if err := scheme.Scheme.Convert(u, apply, nil); err != nil {
				return err
			}

			current := &appsv1.Deployment{}
			if err := scheme.Scheme.Convert(currentU, current, nil); err != nil {
				return err
			}

			// Keep the existing number of replicas in the cluster for the deployment
			apply.Spec.Replicas = current.Spec.Replicas

			// Preserve the env vars in the existing cluster
			for index := range current.Spec.Template.Spec.Containers {
				currentContainer := current.Spec.Template.Spec.Containers[index]
				found, containerIndex := nameExistsContainers(currentContainer.Name, apply.Spec.Template.Spec.Containers)
				if found {
					applyContainer := &apply.Spec.Template.Spec.Containers[containerIndex]
					mergedEnv := currentContainer.Env
					for _, env := range applyContainer.Env {
						found, envIndex := nameExistsEnvVars(env.Name, mergedEnv)
						if !found {
							// Add the new env var into the existing env vars, if it is not available in the existing
							// cluster.
							mergedEnv = append(mergedEnv, env)
						} else if !envVarNames.Has(env.Name) {
							// Set the env var value, if it is available in the existing
							// cluster, but it is not in the preserved list of the env vars.
							mergedEnv[envIndex] = env
						}
					}

					// Remove the env var, which is not in the preserved list and not in the applied manifests
					cleanedMergedEnv := mergedEnv
					for _, env := range mergedEnv {
						found, _ := nameExistsEnvVars(env.Name, applyContainer.Env)
						if !found && !envVarNames.Has(env.Name) {
							// Remove the env var from the cleaned merged env
							cleanedMergedEnv = removeEnvVar(env.Name, cleanedMergedEnv)
						}
					}
					applyContainer.Env = cleanedMergedEnv
				}
			}

			if err := scheme.Scheme.Convert(apply, u, nil); err != nil {
				return err
			}
			// The zero-value timestamp defaulted by the conversion causes
			// superfluous updates
			u.SetCreationTimestamp(metav1.Time{})
		}
		return nil
	}
}

func removeEnvVar(name string, envvars []corev1.EnvVar) []corev1.EnvVar {
	newEnvVars := make([]corev1.EnvVar, 0)
	for _, env := range envvars {
		if env.Name != name {
			newEnvVars = append(newEnvVars, env)
		}
	}
	return newEnvVars
}

func nameExistsEnvVars(name string, envvars []corev1.EnvVar) (bool, int) {
	for index, env := range envvars {
		if env.Name == name {
			return true, index
		}
	}
	return false, -1
}

func nameExistsContainers(name string, containers []corev1.Container) (bool, int) {
	for index, container := range containers {
		if container.Name == name {
			return true, index
		}
	}
	return false, -1
}
