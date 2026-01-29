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
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"knative.dev/operator/pkg/apis/operator/base"
	"sigs.k8s.io/yaml"

	mf "github.com/manifestival/manifestival"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"knative.dev/operator/pkg/apis/operator/v1beta1"
)

// localGateway defines the structure for the entries in the 'local-gateways' array.
type localGatewayConfig struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Service   string `json:"service"`
}

// IngressServiceTransform pins the namespace to istio-system for the service named knative-local-gateway.
// It also removes the OwnerReference to the operator, as they are in different namespaces, which is
// invalid in Kubernetes 1.20+.
func IngressServiceTransform(ks *v1beta1.KnativeServing) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if u.GetAPIVersion() == "v1" && u.GetKind() == "Service" {
			if u.GetName() == "knative-local-gateway" {
				// Default to istio-system, then override if config exists
				u.SetNamespace("istio-system")
				u.SetOwnerReferences(nil)
				config := ks.GetSpec().GetConfig()
				if data, ok := config["istio"]; ok {
					UpdateNamespace(u, data, ks.GetNamespace())
				}

				// The "config-" prefix is optional
				if data, ok := config["config-istio"]; ok {
					UpdateNamespace(u, data, ks.GetNamespace())
				}

				return updateIstioService(ks, u, localGateway)
			}
			if u.GetName() == "knative-ingress-gateway" {
				return updateIstioService(ks, u, ingressGateway)
			}
		}
		return nil
	}
}

func updateIstioService(ks *v1beta1.KnativeServing, u *unstructured.Unstructured,
	ingressGateway func(instance *v1beta1.KnativeServing) *base.IstioGatewayOverride) error {
	service := &corev1.Service{}
	if err := scheme.Scheme.Convert(u, service, nil); err != nil {
		return err
	}
	override := ingressGateway(ks)
	if override != nil && len(override.Selector) > 0 {
		service.Spec.Selector = override.Selector
	}
	if err := scheme.Scheme.Convert(service, u, nil); err != nil {
		return err
	}
	return nil
}

var (
	knativeLocalGateway = "knative-local-gateway"
	localGateways       = "local-gateways"
)

// UpdateNamespace set correct namespace of istio to the service knative-local-gateway
func UpdateNamespace(u *unstructured.Unstructured, data map[string]string, namespace string) {
	if ns := resolveGatewayNamespace(data, namespace); ns != "" {
		u.SetNamespace(ns)
	}
}

func resolveGatewayNamespace(data map[string]string, ns string) string {
	if ns := fromStructuredGateways(data); ns != "" {
		return ns
	}

	return fromLegacyGateway(data, ns)
}

func fromStructuredGateways(data map[string]string) string {
	raw, ok := data[localGateways]
	if !ok {
		return ""
	}

	var gateways []localGatewayConfig
	if err := yaml.Unmarshal([]byte(raw), &gateways); err != nil {
		return ""
	}

	for _, gw := range gateways {
		if gw.Name == knativeLocalGateway {
			return gw.Namespace
		}
	}

	return ""
}

func fromLegacyGateway(data map[string]string, ns string) string {
	key := fmt.Sprintf("local-gateway.%s.%s", ns, knativeLocalGateway)

	raw, ok := data[key]
	if !ok {
		return ""
	}

	fields := strings.Split(raw, ".")
	if len(fields) < 2 {
		return ""
	}

	return fields[1]
}
