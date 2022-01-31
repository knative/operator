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
	"fmt"
	"strings"

	"knative.dev/operator/pkg/apis/operator/base"

	mf "github.com/manifestival/manifestival"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
	caching "knative.dev/caching/pkg/apis/caching/v1alpha1"
)

func init() {
	caching.AddToScheme(scheme.Scheme)
}

var (
	// The string to be replaced by the container name
	containerNameVariable = "${NAME}"
	delimiter             = "/"
)

// ImageTransform updates image with a new registry and tag
func ImageTransform(registry *base.Registry, log *zap.SugaredLogger) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		// Image resources are handled quite differently, so branch them out early
		if u.GetKind() == "Image" && u.GetAPIVersion() == "caching.internal.knative.dev/v1alpha1" {
			return updateCachingImage(registry, u, log)
		}

		// Handle all resources that contain a PodSpec.
		var podSpec *corev1.PodSpec
		var obj metav1.Object

		switch u.GetKind() {
		case "Deployment":
			deployment := &appsv1.Deployment{}
			if err := scheme.Scheme.Convert(u, deployment, nil); err != nil {
				return fmt.Errorf("failed to convert Unstructured to Deployment: %w", err)
			}

			obj = deployment
			podSpec = &deployment.Spec.Template.Spec
		case "DaemonSet":
			ds := &appsv1.DaemonSet{}
			if err := scheme.Scheme.Convert(u, ds, nil); err != nil {
				return fmt.Errorf("failed to convert Unstructured to DaemonSet: %w", err)
			}

			obj = ds
			podSpec = &ds.Spec.Template.Spec
		case "Job":
			job := &batchv1.Job{}
			if err := scheme.Scheme.Convert(u, job, nil); err != nil {
				return fmt.Errorf("failed to convert Unstructured to Job: %w", err)
			}

			obj = job
			podSpec = &job.Spec.Template.Spec
		default:
			// No matches, exit early
			return nil
		}

		log.Debugw("Updating", "name", obj.GetName(), "registry", registry)

		containers := podSpec.Containers
		for i := range containers {
			container := &containers[i]

			// Replace direct image YAML references.
			if image, ok := registry.Override[obj.GetName()+delimiter+container.Name]; ok {
				container.Image = image
			} else if image, ok := registry.Override[container.Name]; ok {
				container.Image = image
			} else if registry.Default != "" {
				// No matches found. Use default setting and replace potential container name placeholder.
				container.Image = strings.ReplaceAll(registry.Default, containerNameVariable, container.Name)
			}

			for j := range container.Env {
				env := &container.Env[j]
				if image, ok := registry.Override[env.Name]; ok {
					env.Value = image
				}
			}
		}

		// Add potential ImagePullSecrets
		if len(registry.ImagePullSecrets) > 0 {
			log.Debugf("Adding ImagePullSecrets: %v", registry.ImagePullSecrets)
			podSpec.ImagePullSecrets = append(podSpec.ImagePullSecrets, registry.ImagePullSecrets...)
		}

		if err := scheme.Scheme.Convert(obj, u, nil); err != nil {
			return err
		}

		// The zero-value timestamp defaulted by the conversion causes
		// superfluous updates
		u.SetCreationTimestamp(metav1.Time{})

		log.Debugw("Finished conversion", "name", u.GetName(), "unstructured", u.Object)
		return nil
	}
}

func updateCachingImage(registry *base.Registry, u *unstructured.Unstructured, log *zap.SugaredLogger) error {
	var img = &caching.Image{}
	if err := scheme.Scheme.Convert(u, img, nil); err != nil {
		return fmt.Errorf("failed to convert Unstructured to Image: %w", err)
	}

	log.Debugw("Updating Image", "name", u.GetName(), "registry", registry)

	// Replace direct image YAML references.
	if image, ok := registry.Override[img.Name]; ok {
		img.Spec.Image = image
	} else if registry.Default != "" {
		// No matches found. Use default setting and replace potential container name placeholder.
		img.Spec.Image = strings.ReplaceAll(registry.Default, containerNameVariable, img.Name)
	}

	// Add potential ImagePullSecrets
	if len(registry.ImagePullSecrets) > 0 {
		log.Debugf("Adding ImagePullSecrets: %v", registry.ImagePullSecrets)
		img.Spec.ImagePullSecrets = append(img.Spec.ImagePullSecrets, registry.ImagePullSecrets...)
	}

	if err := scheme.Scheme.Convert(img, u, nil); err != nil {
		return err
	}
	// Cleanup zero-value default to prevent superfluous updates
	u.SetCreationTimestamp(metav1.Time{})
	delete(u.Object, "status")

	log.Debugw("Finished conversion", "name", u.GetName(), "unstructured", u.Object)
	return nil
}
