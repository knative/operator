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

package ingress

import (
	"context"

	mf "github.com/manifestival/manifestival"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
	"knative.dev/pkg/logging"
)

var noIstio = ingressFilter("istio")

func istioTransformers(ctx context.Context, instance *v1alpha1.KnativeServing) []mf.Transformer {
	logger := logging.FromContext(ctx)
	return []mf.Transformer{gatewayTransform(instance, logger)}
}

func gatewayTransform(instance *v1alpha1.KnativeServing, log *zap.SugaredLogger) mf.Transformer {
	var knativeIngressGateway v1alpha1.IstioGatewayOverride
	var clusterLocalGateway v1alpha1.IstioGatewayOverride

	if instance.Spec.Ingress == nil {
		// Backwards compat. If users don't use ingress API, respect the top-level fields.
		knativeIngressGateway = instance.Spec.KnativeIngressGateway
		clusterLocalGateway = instance.Spec.ClusterLocalGateway
	} else {
		knativeIngressGateway = instance.Spec.Ingress.Istio.KnativeIngressGateway
		clusterLocalGateway = instance.Spec.Ingress.Istio.ClusterLocalGateway
	}

	return func(u *unstructured.Unstructured) error {
		// Update the deployment with the new registry and tag
		if u.GetAPIVersion() == "networking.istio.io/v1alpha3" && u.GetKind() == "Gateway" {
			if u.GetName() == "knative-ingress-gateway" {
				return updateKnativeIngressGateway(knativeIngressGateway, u, log)
			}
			if u.GetName() == "cluster-local-gateway" {
				return updateKnativeIngressGateway(clusterLocalGateway, u, log)
			}
		}
		return nil
	}
}

func updateKnativeIngressGateway(gatewayOverrides v1alpha1.IstioGatewayOverride, u *unstructured.Unstructured, log *zap.SugaredLogger) error {
	if len(gatewayOverrides.Selector) > 0 {
		log.Debugw("Updating Gateway", "name", u.GetName(), "gatewayOverrides", gatewayOverrides)
		unstructured.SetNestedStringMap(u.Object, gatewayOverrides.Selector, "spec", "selector")
		log.Debugw("Finished conversion", "name", u.GetName(), "unstructured", u.Object)
	}
	return nil
}
