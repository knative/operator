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
	mf "github.com/manifestival/manifestival"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	servingv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
	"knative.dev/operator/pkg/reconciler/common"
)

// GatewayTransform is the function that transforms the ingress gateways based on the CR configurations for
// knative-ingress-gateway and cluster-local-gateway
func GatewayTransform(instance *servingv1alpha1.KnativeServing, log *zap.SugaredLogger, manifestPolicy *common.ManifestWithPolicy) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		// Update the deployment with the new registry and tag
		if u.GetAPIVersion() == "networking.istio.io/v1alpha3" && u.GetKind() == "Gateway" {
			if u.GetName() == "knative-ingress-gateway" {
				return updateKnativeIngressGateway(instance.Spec.KnativeIngressGateway, u, log, manifestPolicy)
			}
			if u.GetName() == "cluster-local-gateway" {
				return updateKnativeIngressGateway(instance.Spec.ClusterLocalGateway, u, log, manifestPolicy)
			}
		}
		return nil
	}
}

func updateKnativeIngressGateway(gatewayOverrides servingv1alpha1.IstioGatewayOverride, u *unstructured.Unstructured, log *zap.SugaredLogger,
	manifestPolicy *common.ManifestWithPolicy) (error) {
	if len(gatewayOverrides.Selector) > 0 {
		// User will replace the default gateway with custom gateway, so there is no need to install the default gateway.
		manifestPolicy.GlobalPredicate = mf.All(manifestPolicy.GlobalPredicate, mf.ByName(u.GetName()))
		log.Debugw("Updating Gateway", "name", u.GetName(), "gatewayOverrides", gatewayOverrides)
		unstructured.SetNestedStringMap(u.Object, gatewayOverrides.Selector, "spec", "selector")
		log.Debugw("Finished conversion", "name", u.GetName(), "unstructured", u.Object)
	}
	return nil
}
