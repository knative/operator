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

package ingress

import (
	"context"
	"fmt"

	mf "github.com/manifestival/manifestival"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes/scheme"
	"knative.dev/operator/pkg/apis/operator/v1beta1"
)

const (
	kourierGatewayNSEnvVarKey     = "KOURIER_GATEWAY_NAMESPACE"
	kourierGatewayServiceName     = "kourier"
	kourierDefaultVolumeName      = "kourier-bootstrap"
	kourierGatewayDeploymentNames = "3scale-kourier-gateway"
)

var kourierControllerDeploymentNames = sets.NewString("3scale-kourier-control", "net-kourier-controller")

func kourierTransformers(_ context.Context, instance *v1beta1.KnativeServing) []mf.Transformer {
	return []mf.Transformer{
		replaceGatewayNamespace(),
		configureGatewayService(instance),
		configureBootstrapConfigMap(instance),
	}
}

// replaceGatewayNamespace replace the environment variable KOURIER_GATEWAY_NAMESPACE with the
// namespace of the deployment its set on.
func replaceGatewayNamespace() mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if u.GetKind() == "Deployment" && kourierControllerDeploymentNames.Has(u.GetName()) {
			deployment := &appsv1.Deployment{}
			if err := scheme.Scheme.Convert(u, deployment, nil); err != nil {
				return err
			}

			for i := range deployment.Spec.Template.Spec.Containers {
				c := &deployment.Spec.Template.Spec.Containers[i]
				for j := range c.Env {
					envVar := &c.Env[j]
					if envVar.Name == kourierGatewayNSEnvVarKey {
						envVar.Value = deployment.GetNamespace()
					}
				}
			}

			if err := scheme.Scheme.Convert(deployment, u, nil); err != nil {
				return err
			}
		}
		return nil
	}
}

func configureGatewayService(instance *v1beta1.KnativeServing) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if u.GetKind() != "Service" || u.GetName() != kourierGatewayServiceName {
			return nil
		}

		svc := &v1.Service{}
		if err := scheme.Scheme.Convert(u, svc, nil); err != nil {
			return err
		}

		// First configure the Service Type.
		if instance.Spec.Ingress.Kourier.ServiceType != "" {
			serviceType := instance.Spec.Ingress.Kourier.ServiceType
			switch serviceType {
			case v1.ServiceTypeClusterIP, v1.ServiceTypeLoadBalancer:
				svc.Spec.Type = serviceType
			case v1.ServiceTypeNodePort:
				svc.Spec.Type = serviceType
			case v1.ServiceTypeExternalName:
				return fmt.Errorf("unsupported service type %q", serviceType)
			default:
				return fmt.Errorf("unknown service type %q", serviceType)
			}
		}

		// Configure LoadBalancerIP if set.
		if instance.Spec.Ingress.Kourier.ServiceLoadBalancerIP != "" {
			if svc.Spec.Type != v1.ServiceTypeLoadBalancer {
				return fmt.Errorf("cannot configure LoadBalancerIP for service type %q", svc.Spec.Type)
			}
			svc.Spec.LoadBalancerIP = instance.Spec.Ingress.Kourier.ServiceLoadBalancerIP
		}

		// Configure HTTPPort/HTTPSPort if set.
		if instance.Spec.Ingress.Kourier.HTTPPort > 0 || instance.Spec.Ingress.Kourier.HTTPSPort > 0 {
			if svc.Spec.Type != v1.ServiceTypeNodePort {
				return fmt.Errorf("cannot configure HTTP(S)Port for service type %q", svc.Spec.Type)
			}
			configureGatewayServiceTypeNodePort(instance, svc)
		}

		// Return any conversion error or nil
		return scheme.Scheme.Convert(svc, u, nil)
	}
}

// configureBootstrapConfigMap sets Kourier GW's bootstrap configmap name.
func configureBootstrapConfigMap(instance *v1beta1.KnativeServing) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if u.GetKind() == "Deployment" && u.GetName() == kourierGatewayDeploymentNames {
			if instance.Spec.Ingress.Kourier.BootstrapConfigmapName == "" {
				// Do nothing if BootstrapConfigmapName is not configured.
				return nil
			}
			deployment := &appsv1.Deployment{}
			if err := scheme.Scheme.Convert(u, deployment, nil); err != nil {
				return err
			}

			bootstrapName := instance.Spec.Ingress.Kourier.BootstrapConfigmapName

			for i := range deployment.Spec.Template.Spec.Volumes {
				v := &deployment.Spec.Template.Spec.Volumes[i]
				if v.ConfigMap.Name == kourierDefaultVolumeName {
					v.ConfigMap = &v1.ConfigMapVolumeSource{
						LocalObjectReference: v1.LocalObjectReference{
							Name: bootstrapName,
						},
					}
				}
			}

			if err := scheme.Scheme.Convert(deployment, u, nil); err != nil {
				return err
			}
		}
		return nil
	}
}

func configureGatewayServiceTypeNodePort(instance *v1beta1.KnativeServing, svc *v1.Service) {
	for i, v := range svc.Spec.Ports {
		if v.Name != "https" && instance.Spec.Ingress.Kourier.HTTPPort > 0 {
			v.NodePort = instance.Spec.Ingress.Kourier.HTTPPort
		} else if v.Name == "https" && instance.Spec.Ingress.Kourier.HTTPSPort > 0 {
			v.NodePort = instance.Spec.Ingress.Kourier.HTTPSPort
		}
		svc.Spec.Ports[i] = v
	}
}
