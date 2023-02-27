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
	istionetworkingv1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	"istio.io/client-go/pkg/clientset/versioned/scheme"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"knative.dev/operator/pkg/apis/operator/base"
	"knative.dev/operator/pkg/apis/operator/v1beta1"
	servingv1beta1 "knative.dev/operator/pkg/apis/operator/v1beta1"
	"knative.dev/pkg/logging"
)

//var istioFilter = ingressFilter("istio")

func istioTransformers(ctx context.Context, instance *v1beta1.KnativeServing) []mf.Transformer {
	logger := logging.FromContext(ctx)
	return []mf.Transformer{gatewayTransform(instance, logger)}
}

func gatewayTransform(instance *servingv1beta1.KnativeServing, log *zap.SugaredLogger) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		// Update the deployment with the new registry and tag
		if u.GetAPIVersion() == "networking.istio.io/v1alpha3" && u.GetKind() == "Gateway" {
			gateway := &istionetworkingv1alpha3.Gateway{}
			if err := scheme.Scheme.Convert(u, gateway, nil); err != nil {
				return err
			}

			if u.GetName() == "knative-ingress-gateway" {
				if err := updateIstioGateway(ingressGateway(instance), gateway, log); err != nil {
					return err
				}
			}
			// TODO: cluster-local-gateway was removed since v0.20 https://github.com/knative-sandbox/net-istio/commit/058432d749435ef1fc61aa2b1fd048d0c75460ee
			// Reomove it once operator stops v0.20 support.
			if u.GetName() == "cluster-local-gateway" || u.GetName() == "knative-local-gateway" {
				if err := updateIstioGateway(localGateway(instance), gateway, log); err != nil {
					return err
				}
			}

			if err := scheme.Scheme.Convert(gateway, u, nil); err != nil {
				return err
			}
		}
		return nil
	}
}

func ingressGateway(instance *servingv1beta1.KnativeServing) *base.IstioGatewayOverride {
	if instance.Spec.Ingress != nil && instance.Spec.Ingress.Istio.KnativeIngressGateway != nil {
		return instance.Spec.Ingress.Istio.KnativeIngressGateway
	}
	return nil
}

func localGateway(instance *servingv1beta1.KnativeServing) *base.IstioGatewayOverride {
	if instance.Spec.Ingress != nil && instance.Spec.Ingress.Istio.KnativeLocalGateway != nil {
		return instance.Spec.Ingress.Istio.KnativeLocalGateway
	}
	return nil
}

func updateIstioGateway(override *base.IstioGatewayOverride, gateway *istionetworkingv1alpha3.Gateway, log *zap.SugaredLogger) error {
	if override != nil && len(override.Selector) > 0 {
		log.Debugw("Updating Gateway", "name", gateway.GetName(), "gatewayOverrides", override)
		gateway.Spec.Selector = override.Selector
		log.Debugw("Finished conversion", "name", gateway.GetName())
	}

	if override != nil && len(override.Servers) > 0 {
		log.Debugw("Updating Gateway Servers", "name", gateway.GetName(), "gatewayOverrides", override)
		gateway.Spec.Servers = override.Servers
		log.Debugw("Finished Servers Overrides", "name", gateway.GetName())
	}
	return nil
}
