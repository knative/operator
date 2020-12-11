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
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
)

type unstructuredGetter interface {
	Get(obj *unstructured.Unstructured) (*unstructured.Unstructured, error)
}

// DispatcherAdapterTransform keeps the number of replicas and the env vars, if the deployment
// pingsource-mt-adapter, kafka-ch-dispatcher or imc-dispatcher exists in the cluster.
func DispatcherAdapterTransform(client unstructuredGetter) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if u.GetKind() == "Deployment" && (u.GetName() == "pingsource-mt-adapter" ||
			u.GetName() == "imc-dispatcher") {
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

			if u.GetName() == "pingsource-mt-adapter" {
				// Copy the existing env vars of existing containers
				for index := range current.Spec.Template.Spec.Containers {
					currentContainer := current.Spec.Template.Spec.Containers[index]
					for j := range apply.Spec.Template.Spec.Containers {
						applyContainer := &apply.Spec.Template.Spec.Containers[j]
						if currentContainer.Name == applyContainer.Name {
							applyContainer.Env = currentContainer.Env
						}
					}
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
