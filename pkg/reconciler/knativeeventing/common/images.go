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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"

	eventingv1alpha1 "knative.dev/operator/pkg/apis/eventing/v1alpha1"
)

var (
	// The string to be replaced by the container name
	containerNameVariable = "${NAME}"
	delimiter             = "/"
)

// DeploymentTransform updates the links of images with customized registries for all deployments
func DeploymentTransform(instance *eventingv1alpha1.KnativeEventing, log *zap.SugaredLogger) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		// Update the deployment with the new registry and tag
		if u.GetKind() == "Deployment" {
			return updateDeployment(instance, u, log)
		}
		return nil
	}
}

func updateDeployment(instance *eventingv1alpha1.KnativeEventing, u *unstructured.Unstructured, log *zap.SugaredLogger) error {
	var deployment = &appsv1.Deployment{}
	err := scheme.Scheme.Convert(u, deployment, nil)
	if err != nil {
		log.Error(err, "Error converting Unstructured to Deployment", "unstructured", u, "deployment", deployment)
		return err
	}

	registry := instance.Spec.Registry
	log.Debugw("Updating Deployment", "name", u.GetName(), "registry", registry)

	updateDeploymentImage(deployment, &registry, log)
	updateDeploymentEnvVarImages(deployment, &registry, log)

	deployment.Spec.Template.Spec.ImagePullSecrets = addImagePullSecrets(
		deployment.Spec.Template.Spec.ImagePullSecrets, &registry, log)
	err = scheme.Scheme.Convert(deployment, u, nil)
	if err != nil {
		return err
	}
	// The zero-value timestamp defaulted by the conversion causes
	// superfluous updates
	u.SetCreationTimestamp(metav1.Time{})

	log.Debugw("Finished conversion", "name", u.GetName(), "unstructured", u.Object)
	return nil
}

func updateDeploymentEnvVarImages(deployment *appsv1.Deployment, registry *eventingv1alpha1.Registry, log *zap.SugaredLogger) {
	containers := deployment.Spec.Template.Spec.Containers
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

// updateDeploymentImage updates the image of the deployment with a new registry and tag
func updateDeploymentImage(deployment *appsv1.Deployment, registry *eventingv1alpha1.Registry, log *zap.SugaredLogger) {
	containers := deployment.Spec.Template.Spec.Containers
	for index := range containers {
		container := &containers[index]
		newImage := getNewImage(registry, container.Name, deployment.Name)
		if newImage != "" {
			updateContainer(container, newImage, log)
		}
	}
	log.Debugw("Finished updating images", "name", deployment.GetName(), "containers", deployment.Spec.Template.Spec.Containers)
}

func getNewImage(registry *eventingv1alpha1.Registry, containerName, deploymentName string) string {
	overrideImage := registry.Override[deploymentName+delimiter+containerName]
	if overrideImage == "" {
		overrideImage = registry.Override[containerName]
	}
	if overrideImage != "" {
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

func addImagePullSecrets(imagePullSecrets []corev1.LocalObjectReference, registry *eventingv1alpha1.Registry, log *zap.SugaredLogger) []corev1.LocalObjectReference {
	if len(registry.ImagePullSecrets) > 0 {
		log.Debugf("Adding ImagePullSecrets: %v", registry.ImagePullSecrets)
		imagePullSecrets = append(imagePullSecrets, registry.ImagePullSecrets...)
	}
	return imagePullSecrets
}
