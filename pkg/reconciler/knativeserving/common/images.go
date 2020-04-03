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
package common

import (
	"strings"

	mf "github.com/manifestival/manifestival"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
	caching "knative.dev/caching/pkg/apis/caching/v1alpha1"
	servingv1alpha1 "knative.dev/serving-operator/pkg/apis/serving/v1alpha1"
)

func init() {
	caching.AddToScheme(scheme.Scheme)
}

var (
	// The string to be replaced by the container name
	containerNameVariable = "${NAME}"
)

// ImageTransform updates image with a new registry and tag
func ImageTransform(instance *servingv1alpha1.KnativeServing, log *zap.SugaredLogger) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		switch u.GetKind() {
		// TODO need to use PodSpecable duck type in order to remove duplicates of deployment, daemonSet
		case "Deployment":
			return updateDeployment(instance, u, log)
		case "DaemonSet":
			return updateDaemonSet(instance, u, log)
		case "Image":
			if u.GetAPIVersion() == "caching.internal.knative.dev/v1alpha1" {
				return updateCachingImage(instance, u)
			}
		}
		return nil
	}
}

func updateDeployment(instance *servingv1alpha1.KnativeServing, u *unstructured.Unstructured, log *zap.SugaredLogger) error {
	var deployment = &appsv1.Deployment{}
	if err := scheme.Scheme.Convert(u, deployment, nil); err != nil {
		log.Error(err, "Error converting Unstructured to Deployment", "unstructured", u, "deployment", deployment)
		return err
	}

	updateRegistry(&deployment.Spec.Template.Spec, instance, log, deployment.GetName())
	if err := scheme.Scheme.Convert(deployment, u, nil); err != nil {
		return err
	}
	// The zero-value timestamp defaulted by the conversion causes
	// superfluous updates
	u.SetCreationTimestamp(metav1.Time{})

	log.Debugw("Finished conversion", "name", u.GetName(), "unstructured", u.Object)
	return nil
}

func updateDaemonSet(instance *servingv1alpha1.KnativeServing, u *unstructured.Unstructured, log *zap.SugaredLogger) error {
	var daemonSet = &appsv1.DaemonSet{}
	if err := scheme.Scheme.Convert(u, daemonSet, nil); err != nil {
		log.Error(err, "Error converting Unstructured to daemonSet", "unstructured", u, "daemonSet", daemonSet)
		return err
	}
	updateRegistry(&daemonSet.Spec.Template.Spec, instance, log, daemonSet.GetName())
	if err := scheme.Scheme.Convert(daemonSet, u, nil); err != nil {
		return err
	}
	// The zero-value timestamp defaulted by the conversion causes
	// superfluous updates
	u.SetCreationTimestamp(metav1.Time{})

	log.Debugw("Finished conversion", "name", u.GetName(), "unstructured", u.Object)
	return nil
}

func updateRegistry(spec *corev1.PodSpec, instance *servingv1alpha1.KnativeServing, log *zap.SugaredLogger, name string) {
	registry := instance.Spec.Registry
	log.Debugw("Updating", "name", name, "registry", registry)

	updateImage(spec, &registry, log, name)
	spec.ImagePullSecrets = addImagePullSecrets(
		spec.ImagePullSecrets, &registry, log)
}

// updateImage updates the image with a new registry and tag
func updateImage(spec *corev1.PodSpec, registry *servingv1alpha1.Registry, log *zap.SugaredLogger, name string) {
	containers := spec.Containers
	for index := range containers {
		container := &containers[index]
		newImage := getNewImage(registry, container.Name)
		if newImage != "" {
			updateContainer(container, newImage, log)
		}
	}
	log.Debugw("Finished updating images", "name", name, "containers", spec.Containers)
}

func updateCachingImage(instance *servingv1alpha1.KnativeServing, u *unstructured.Unstructured) error {
	var image = &caching.Image{}
	if err := scheme.Scheme.Convert(u, image, nil); err != nil {
		log.Error(err, "Error converting Unstructured to Image", "unstructured", u, "image", image)
		return err
	}

	registry := instance.Spec.Registry
	log.Debugw("Updating Image", "name", u.GetName(), "registry", registry)

	updateImageSpec(image, &registry, log)
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
func updateImageSpec(image *caching.Image, registry *servingv1alpha1.Registry, log *zap.SugaredLogger) {
	if newImage := getNewImage(registry, image.Name); newImage != "" {
		log.Debugf("Updating image from: %v, to: %v", image.Spec.Image, newImage)
		image.Spec.Image = newImage
	}
	image.Spec.ImagePullSecrets = addImagePullSecrets(image.Spec.ImagePullSecrets, registry, log)
	log.Debugw("Finished updating image", "image", image.GetName())
}

func getNewImage(registry *servingv1alpha1.Registry, containerName string) string {
	if overrideImage := registry.Override[containerName]; overrideImage != "" {
		return overrideImage
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

func addImagePullSecrets(imagePullSecrets []corev1.LocalObjectReference, registry *servingv1alpha1.Registry, log *zap.SugaredLogger) []corev1.LocalObjectReference {
	if len(registry.ImagePullSecrets) > 0 {
		log.Debugf("Adding ImagePullSecrets: %v", registry.ImagePullSecrets)
		imagePullSecrets = append(imagePullSecrets, registry.ImagePullSecrets...)
	}
	return imagePullSecrets
}
