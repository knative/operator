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
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
	caching "knative.dev/caching/pkg/apis/caching/v1alpha1"
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
)

func init() {
	caching.AddToScheme(scheme.Scheme)
}

var (
	// The string to be replaced by the container name
	containerNameVariable = "${NAME}"
	delimiter             = "/"
)

// ImageTransformer is an interface for transforming images passed to the ResourceImageTransformer
type ImageTransformer interface {
	ImageForContainer(container *corev1.Container, parentName string) (string, bool)
	ImageForEnvVar(env *corev1.EnvVar, parentName string) (string, bool)
	ImageForImage(image *caching.Image, parentName string) (string, bool)
	HandleImagePullSecrets(imagePullSecrets []corev1.LocalObjectReference, log *zap.SugaredLogger) []corev1.LocalObjectReference
}

// registryImageTransformer is a v1alpha1.Registry specific transformer
type registryImageTransformer struct {
	registry *v1alpha1.Registry
}

var _ ImageTransformer = (*registryImageTransformer)(nil)

func (rit *registryImageTransformer) ImageForContainer(container *corev1.Container, parentName string) (string, bool) {
	return rit.handleImage(container.Name, parentName, true)
}

func (rit *registryImageTransformer) ImageForEnvVar(env *corev1.EnvVar, parentName string) (string, bool) {
	return rit.handleImage(env.Name, "", false)
}

func (rit *registryImageTransformer) ImageForImage(image *caching.Image, parentName string) (string, bool) {
	return rit.handleImage(image.Name, "", true)
}

func (rit *registryImageTransformer) handleImage(resourceName, parentName string, useDefault bool) (string, bool) {
	if image, ok := rit.registry.Override[parentName+delimiter+resourceName]; ok {
		return image, true
	}
	if image, ok := rit.registry.Override[resourceName]; ok {
		return image, true
	}
	if !useDefault {
		return "", false
	}
	return replaceName(rit.registry.Default, resourceName), true
}

func replaceName(imageTemplate string, name string) string {
	return strings.ReplaceAll(imageTemplate, containerNameVariable, name)
}

func (rit *registryImageTransformer) HandleImagePullSecrets(imagePullSecrets []corev1.LocalObjectReference, log *zap.SugaredLogger) []corev1.LocalObjectReference {
	if len(rit.registry.ImagePullSecrets) > 0 {
		log.Debugf("Adding ImagePullSecrets: %v", rit.registry.ImagePullSecrets)
		imagePullSecrets = append(imagePullSecrets, rit.registry.ImagePullSecrets...)
	}
	return imagePullSecrets
}

// ImageTransform updates image with a new registry and tag
func ImageTransform(registry *v1alpha1.Registry, log *zap.SugaredLogger) mf.Transformer {
	rit := &registryImageTransformer{
		registry: registry,
	}
	return ResourceImageTransformer(rit, log)
}

// ResourceImageTransformer takes an ImageTransformer and transform images across resources
func ResourceImageTransformer(imageTransformer ImageTransformer, log *zap.SugaredLogger) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		switch u.GetKind() {
		// TODO need to use PodSpecable duck type in order to remove duplicates of deployment, daemonSet
		case "Deployment":
			return updateDeployment(imageTransformer, u, log)
		case "DaemonSet":
			return updateDaemonSet(imageTransformer, u, log)
		case "Job":
			return updateJob(imageTransformer, u, log)
		case "Image":
			if u.GetAPIVersion() == "caching.internal.knative.dev/v1alpha1" {
				return updateCachingImage(imageTransformer, u, log)
			}
		}
		return nil
	}
}

func updateDeployment(imageTransformer ImageTransformer, u *unstructured.Unstructured, log *zap.SugaredLogger) error {
	var deployment = &appsv1.Deployment{}
	if err := scheme.Scheme.Convert(u, deployment, nil); err != nil {
		log.Error(err, "Error converting Unstructured to Deployment", "unstructured", u, "deployment", deployment)
		return err
	}

	updateRegistry(&deployment.Spec.Template.Spec, imageTransformer, log, deployment.GetName())
	if err := scheme.Scheme.Convert(deployment, u, nil); err != nil {
		return err
	}
	// The zero-value timestamp defaulted by the conversion causes
	// superfluous updates
	u.SetCreationTimestamp(metav1.Time{})

	log.Debugw("Finished conversion", "name", u.GetName(), "unstructured", u.Object)
	return nil
}

func updateDaemonSet(imageTransformer ImageTransformer, u *unstructured.Unstructured, log *zap.SugaredLogger) error {
	var daemonSet = &appsv1.DaemonSet{}
	if err := scheme.Scheme.Convert(u, daemonSet, nil); err != nil {
		log.Error(err, "Error converting Unstructured to daemonSet", "unstructured", u, "daemonSet", daemonSet)
		return err
	}
	updateRegistry(&daemonSet.Spec.Template.Spec, imageTransformer, log, daemonSet.GetName())
	if err := scheme.Scheme.Convert(daemonSet, u, nil); err != nil {
		return err
	}
	// The zero-value timestamp defaulted by the conversion causes
	// superfluous updates
	u.SetCreationTimestamp(metav1.Time{})

	log.Debugw("Finished conversion", "name", u.GetName(), "unstructured", u.Object)
	return nil
}

func updateJob(imageTransformer ImageTransformer, u *unstructured.Unstructured, log *zap.SugaredLogger) error {
	var job = &batchv1.Job{}
	if err := scheme.Scheme.Convert(u, job, nil); err != nil {
		log.Error(err, "Error converting Unstructured to job", "unstructured", u, "job", job)
		return err
	}
	updateRegistry(&job.Spec.Template.Spec, imageTransformer, log, job.GetName())
	if err := scheme.Scheme.Convert(job, u, nil); err != nil {
		return err
	}
	// The zero-value timestamp defaulted by the conversion causes
	// superfluous updates
	u.SetCreationTimestamp(metav1.Time{})

	log.Debugw("Finished conversion", "name", u.GetName(), "unstructured", u.Object)
	return nil
}

func updateRegistry(spec *corev1.PodSpec, imageTransformer ImageTransformer, log *zap.SugaredLogger, name string) {
	log.Debugw("Updating", "name", name, "imageTransformer", imageTransformer)

	updateImage(spec, imageTransformer, log, name)
	updateEnvVarImages(spec, imageTransformer, log, name)

	spec.ImagePullSecrets = imageTransformer.HandleImagePullSecrets(
		spec.ImagePullSecrets, log)
}

// updateImage updates the image with a new registry and tag
func updateImage(spec *corev1.PodSpec, imageTransformer ImageTransformer, log *zap.SugaredLogger, name string) {
	containers := spec.Containers
	for index := range containers {
		container := &containers[index]
		newImage, _ := imageTransformer.ImageForContainer(container, name)
		if newImage != "" {
			updateContainer(container, newImage, log)
		}
	}
	log.Debugw("Finished updating images", "name", name, "containers", spec.Containers)
}

func updateEnvVarImages(spec *corev1.PodSpec, imageTransformer ImageTransformer, log *zap.SugaredLogger, name string) {
	containers := spec.Containers
	for index := range containers {
		container := &containers[index]
		for envIndex := range container.Env {
			env := &container.Env[envIndex]
			if newImage, ok := imageTransformer.ImageForEnvVar(env, container.Name); ok {
				env.Value = newImage
			}
		}
	}
}

func updateCachingImage(imageTransformer ImageTransformer, u *unstructured.Unstructured, log *zap.SugaredLogger) error {
	var image = &caching.Image{}
	if err := scheme.Scheme.Convert(u, image, nil); err != nil {
		log.Error(err, "Error converting Unstructured to Image", "unstructured", u, "image", image)
		return err
	}

	log.Debugw("Updating Image", "name", u.GetName(), "registry", imageTransformer)

	updateImageSpec(image, imageTransformer, log)
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
func updateImageSpec(image *caching.Image, imageTransformer ImageTransformer, log *zap.SugaredLogger) {
	if newImage, _ := imageTransformer.ImageForImage(image, ""); newImage != "" {
		log.Debugf("Updating image from: %v, to: %v", image.Spec.Image, newImage)
		image.Spec.Image = newImage
	}
	image.Spec.ImagePullSecrets = imageTransformer.HandleImagePullSecrets(image.Spec.ImagePullSecrets, log)
	log.Debugw("Finished updating image", "image", image.GetName())
}

func updateContainer(container *corev1.Container, newImage string, log *zap.SugaredLogger) {
	log.Debugf("Updating container image from: %v, to: %v", container.Image, newImage)
	container.Image = newImage
}
