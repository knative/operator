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

	"knative.dev/operator/pkg/apis/operator/base"

	mf "github.com/manifestival/manifestival"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
	servingv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
	"knative.dev/pkg/logging"
)

var istioFilter = ingressFilter("istio")

func istioTransformers(ctx context.Context, instance *v1alpha1.KnativeServing) []mf.Transformer {
	logger := logging.FromContext(ctx)
	return []mf.Transformer{gatewayTransform(instance, logger)}
}

func gatewayTransform(instance *servingv1alpha1.KnativeServing, log *zap.SugaredLogger) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		// Update the deployment with the new registry and tag
		if u.GetAPIVersion() == "networking.istio.io/v1alpha3" && u.GetKind() == "Gateway" {
			if u.GetName() == "knative-ingress-gateway" {
				return updateIstioGateway(ingressGateway(instance), u, log)
			}
			// TODO: cluster-local-gateway was removed since v0.20 https://github.com/knative-sandbox/net-istio/commit/058432d749435ef1fc61aa2b1fd048d0c75460ee
			// Reomove it once operator stops v0.20 support.
			if u.GetName() == "cluster-local-gateway" || u.GetName() == "knative-local-gateway" {
				return updateIstioGateway(localGateway(instance), u, log)
			}
		}
		return nil
	}
}

func ingressGateway(instance *servingv1alpha1.KnativeServing) *base.IstioGatewayOverride {
	if instance.Spec.Ingress != nil && instance.Spec.Ingress.Istio.KnativeIngressGateway != nil {
		return instance.Spec.Ingress.Istio.KnativeIngressGateway
	}
	return &instance.Spec.DeprecatedKnativeIngressGateway
}

func localGateway(instance *servingv1alpha1.KnativeServing) *base.IstioGatewayOverride {
	if instance.Spec.Ingress != nil && instance.Spec.Ingress.Istio.KnativeLocalGateway != nil {
		return instance.Spec.Ingress.Istio.KnativeLocalGateway
	}
	return &instance.Spec.DeprecatedClusterLocalGateway
}

func updateIstioGateway(override *base.IstioGatewayOverride, u *unstructured.Unstructured, log *zap.SugaredLogger) error {
	if override != nil && len(override.Selector) > 0 {
		log.Debugw("Updating Gateway", "name", u.GetName(), "gatewayOverrides", override)
		unstructured.SetNestedStringMap(u.Object, override.Selector, "spec", "selector")
		log.Debugw("Finished conversion", "name", u.GetName(), "unstructured", u.Object)
	}
	return nil
}
