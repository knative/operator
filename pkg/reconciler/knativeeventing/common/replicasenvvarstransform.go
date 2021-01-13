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
				applyContainer := findContainer(currentContainer.Name, apply.Spec.Template.Spec.Containers)
				if applyContainer == nil {
					continue
				}
				var mergedEnv []corev1.EnvVar
				actualKeys := sets.NewString()
				for _, env := range currentContainer.Env {
					if envVarNames.Has(env.Name) {
						// Keep all the env vars in the preserved list
						mergedEnv = append(mergedEnv, env)
						actualKeys.Insert(env.Name)
					}
				}

				for _, env := range applyContainer.Env {
					if !actualKeys.Has(env.Name) {
						// Apply all keys that are neither preserved, nor in the actual container.
						mergedEnv = append(mergedEnv, env)
					}
				}

				applyContainer.Env = mergedEnv
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

func findContainer(name string, containers []corev1.Container) *corev1.Container {
	for index, container := range containers {
		if container.Name == name {
			return &containers[index]
		}
	}
	return nil
}
