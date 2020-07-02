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
	"fmt"

	mf "github.com/manifestival/manifestival/pkg/transform"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
	servingv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
)

const (
	customCertsEnvName    = "SSL_CERT_DIR"
	customCertsMountPath  = "/custom-certs"
	customCertsNamePrefix = "custom-certs-"
)

// CustomCertsTransform configures the controller deployment to trust
// registries with self-signed certs
func CustomCertsTransform(instance *servingv1alpha1.KnativeServing, log *zap.SugaredLogger) mf.Transformer {
	empty := servingv1alpha1.CustomCerts{}
	return func(u *unstructured.Unstructured) error {
		if instance.Spec.ControllerCustomCerts == empty {
			return nil
		}
		if u.GetKind() == "Deployment" && u.GetName() == "controller" {
			certs := instance.Spec.ControllerCustomCerts
			deployment := &appsv1.Deployment{}
			if err := scheme.Scheme.Convert(u, deployment, nil); err != nil {
				return err
			}
			if err := configureCustomCerts(deployment, certs); err != nil {
				return err
			}
			if err := scheme.Scheme.Convert(deployment, u, nil); err != nil {
				return err
			}
			// Avoid superfluous updates from converted zero defaults
			u.SetCreationTimestamp(metav1.Time{})
		}
		return nil
	}
}

func configureCustomCerts(deployment *appsv1.Deployment, certs servingv1alpha1.CustomCerts) error {
	source := v1.VolumeSource{}
	switch certs.Type {
	case "ConfigMap":
		source.ConfigMap = &v1.ConfigMapVolumeSource{
			LocalObjectReference: v1.LocalObjectReference{
				Name: certs.Name,
			},
		}
	case "Secret":
		source.Secret = &v1.SecretVolumeSource{
			SecretName: certs.Name,
		}
	default:
		return fmt.Errorf("Unknown CustomCerts type: %s", certs.Type)
	}

	name := customCertsNamePrefix + certs.Name
	if name == customCertsNamePrefix {
		return fmt.Errorf("CustomCerts name for %s is required", certs.Type)
	}
	deployment.Spec.Template.Spec.Volumes = append(deployment.Spec.Template.Spec.Volumes, v1.Volume{
		Name:         name,
		VolumeSource: source,
	})

	containers := deployment.Spec.Template.Spec.Containers
	containers[0].VolumeMounts = append(containers[0].VolumeMounts, v1.VolumeMount{
		Name:      name,
		MountPath: customCertsMountPath,
	})
	containers[0].Env = append(containers[0].Env, v1.EnvVar{
		Name:  customCertsEnvName,
		Value: customCertsMountPath,
	})
	return nil
}
