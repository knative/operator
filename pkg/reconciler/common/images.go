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
	"strings"

	mf "github.com/manifestival/manifestival"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
	caching "knative.dev/caching/pkg/apis/caching/v1alpha1"
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
	duckv1 "knative.dev/pkg/apis/duck/v1"
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
func ImageTransform(registry *v1alpha1.Registry, log *zap.SugaredLogger) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		switch u.GetKind() {
		case "Deployment":
			fallthrough
		case "DaemonSet":
			fallthrough
		case "Job":
			return updatePodSpecable(registry, u, log)
		case "Image":
			if u.GetAPIVersion() == "caching.internal.knative.dev/v1alpha1" {
				return updateCachingImage(registry, u, log)
			}
		}
		return nil
	}
}

func updatePodSpecable(registry *v1alpha1.Registry, u *unstructured.Unstructured, log *zap.SugaredLogger) error {
	var withPod = &duckv1.WithPod{}
	if err := scheme.Scheme.Convert(u, withPod, nil); err != nil {
		log.Error(err, "Error converting Unstructured to Deployment", "unstructured", u, "withPod", withPod)
		return err
	}

	updateRegistry(&withPod.Spec.Template.Spec, registry, log, withPod.GetName())
	if err := scheme.Scheme.Convert(withPod, u, nil); err != nil {
		return err
	}
	// The zero-value timestamp defaulted by the conversion causes
	// superfluous updates
	u.SetCreationTimestamp(metav1.Time{})

	log.Debugw("Finished conversion", "name", u.GetName(), "unstructured", u.Object)
	return nil
}

func updateRegistry(spec *corev1.PodSpec, registry *v1alpha1.Registry, log *zap.SugaredLogger, name string) {
	log.Debugw("Updating", "name", name, "registry", registry)

	updateImage(spec, registry, log, name)
	updateEnvVarImages(spec, registry, log, name)

	spec.ImagePullSecrets = addImagePullSecrets(
		spec.ImagePullSecrets, registry, log)
}

// updateImage updates the image with a new registry and tag
func updateImage(spec *corev1.PodSpec, registry *v1alpha1.Registry, log *zap.SugaredLogger, name string) {
	containers := spec.Containers
	for index := range containers {
		container := &containers[index]
		newImage := getNewImage(registry, container.Name, name)
		if newImage != "" {
			updateContainer(container, newImage, log)
		}
	}
	log.Debugw("Finished updating images", "name", name, "containers", spec.Containers)
}

func updateEnvVarImages(spec *corev1.PodSpec, registry *v1alpha1.Registry, log *zap.SugaredLogger, name string) {
	containers := spec.Containers
	for index := range containers {
		container := &containers[index]
		for envIndex := range container.Env {
			env := &container.Env[envIndex]
			if newImage, ok := registry.Override[env.Name]; ok {
				env.Value = newImage
			}
		}
	}
}

func updateCachingImage(registry *v1alpha1.Registry, u *unstructured.Unstructured, log *zap.SugaredLogger) error {
	var image = &caching.Image{}
	if err := scheme.Scheme.Convert(u, image, nil); err != nil {
		log.Error(err, "Error converting Unstructured to Image", "unstructured", u, "image", image)
		return err
	}

	log.Debugw("Updating Image", "name", u.GetName(), "registry", registry)

	updateImageSpec(image, registry, log)
	if err := scheme.Scheme.Convert(image, u, nil); err != nil {
		return err
	}
	// Cleanup zero-value default to prevent superfluous updates
	u.SetCreationTimestamp(metav1.Time{})
	delete(u.Object, "status")

	log.Debugw("Finished conversion", "name", u.GetName(), "unstructured", u.Object)
	return nil
}

// updateImageSpec updates the image of a with a new registry and tag
func updateImageSpec(image *caching.Image, registry *v1alpha1.Registry, log *zap.SugaredLogger) {
	if newImage := getNewImage(registry, image.Name, ""); newImage != "" {
		log.Debugf("Updating image from: %v, to: %v", image.Spec.Image, newImage)
		image.Spec.Image = newImage
	}
	image.Spec.ImagePullSecrets = addImagePullSecrets(image.Spec.ImagePullSecrets, registry, log)
	log.Debugw("Finished updating image", "image", image.GetName())
}

func getNewImage(registry *v1alpha1.Registry, containerName, parent string) string {
	if image, ok := registry.Override[parent+delimiter+containerName]; ok {
		return image
	}
	if image, ok := registry.Override[containerName]; ok {
		return image
	}
	return replaceName(registry.Default, containerName)
}

func updateContainer(container *corev1.Container, newImage string, log *zap.SugaredLogger) {
	log.Debugf("Updating container image from: %v, to: %v", container.Image, newImage)
	container.Image = newImage
}

func replaceName(imageTemplate string, name string) string {
	return strings.ReplaceAll(imageTemplate, containerNameVariable, name)
}

func addImagePullSecrets(imagePullSecrets []corev1.LocalObjectReference, registry *v1alpha1.Registry, log *zap.SugaredLogger) []corev1.LocalObjectReference {
	if len(registry.ImagePullSecrets) > 0 {
		log.Debugf("Adding ImagePullSecrets: %v", registry.ImagePullSecrets)
		imagePullSecrets = append(imagePullSecrets, registry.ImagePullSecrets...)
	}
	return imagePullSecrets
}
