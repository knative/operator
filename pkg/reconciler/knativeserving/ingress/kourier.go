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

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"

	mf "github.com/manifestival/manifestival"
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
)

const (
	kourierGatewayNSEnvVarKey = "KOURIER_GATEWAY_NAMESPACE"
	kourierDeploymentName     = "3scale-kourier-control"
)

var kourierFilter = ingressFilter("kourier")

func kourierTransformers(ctx context.Context, instance *v1alpha1.KnativeServing) []mf.Transformer {
	return []mf.Transformer{
		replaceKourierGWNamespace(instance.GetNamespace()),
	}
}

// replaceKourierGWNamespace replace the environment variable KOURIER_GATEWAY_NAMESPACE with the namespace of the Knative Serving CR
func replaceKourierGWNamespace(ns string) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if u.GetKind() == "Deployment" && u.GetName() == kourierDeploymentName {
			_, hasLabel := u.GetLabels()[providerLabel]
			if !hasLabel {
				return nil
			}
			deployment := &appsv1.Deployment{}
			if err := scheme.Scheme.Convert(u, deployment, nil); err != nil {
				return err
			}
			for i := range deployment.Spec.Template.Spec.Containers {
				c := &deployment.Spec.Template.Spec.Containers[i]
				for j := range c.Env {
					envVar := &c.Env[j]
					if envVar.Name == kourierGatewayNSEnvVarKey {
						envVar.Value = ns
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
