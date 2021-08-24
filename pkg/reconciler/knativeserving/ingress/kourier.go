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
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
)

const (
	kourierGatewayNSEnvVarKey = "KOURIER_GATEWAY_NAMESPACE"
	kourierGatewayServiceName = "kourier"
)

var kourierControllerDeploymentNames = sets.NewString("3scale-kourier-control", "net-kourier-controller")

var kourierFilter = ingressFilter("kourier")

func kourierTransformers(ctx context.Context, instance *v1alpha1.KnativeServing) []mf.Transformer {
	return []mf.Transformer{
		replaceGWNamespace(),
		configureGWServiceType(instance),
	}
}

// replaceGWNamespace replace the environment variable KOURIER_GATEWAY_NAMESPACE with the
// namespace of the deployment its set on.
func replaceGWNamespace() mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if u.GetKind() == "Deployment" && kourierControllerDeploymentNames.Has(u.GetName()) && hasProviderLabel(u) {
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

// configureGWServiceType configures Kourier GW's service type such as ClusterIP, LoadBalancer and NodePort.
func configureGWServiceType(instance *v1alpha1.KnativeServing) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if u.GetKind() == "Service" && u.GetName() == kourierGatewayServiceName && hasProviderLabel(u) {
			if instance.Spec.Ingress.Kourier.ServiceType == "" {
				// Do nothing if ServiceType is not configured.
				return nil
			}
			svc := &v1.Service{}
			if err := scheme.Scheme.Convert(u, svc, nil); err != nil {
				return err
			}

			serviceType := instance.Spec.Ingress.Kourier.ServiceType
			switch serviceType {
			case v1.ServiceTypeClusterIP, v1.ServiceTypeNodePort, v1.ServiceTypeLoadBalancer:
				svc.Spec.Type = serviceType
			case v1.ServiceTypeExternalName:
				return fmt.Errorf("unsupported service type %q", serviceType)
			default:
				return fmt.Errorf("unknown service type %q", serviceType)
			}

			if err := scheme.Scheme.Convert(svc, u, nil); err != nil {
				return err
			}
		}
		return nil
	}
}
