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
	"fmt"
	"strings"

	mf "github.com/manifestival/manifestival"
	"go.uber.org/zap"
	istiov1beta1 "istio.io/api/networking/v1beta1"
	istionetworkingv1beta "istio.io/client-go/pkg/apis/networking/v1beta1"
	"istio.io/client-go/pkg/clientset/versioned/scheme"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"knative.dev/operator/pkg/apis/operator/base"
	servingv1beta1 "knative.dev/operator/pkg/apis/operator/v1beta1"
	"knative.dev/pkg/logging"
)

// istioTLSModes maps the string TLS mode values accepted on the operator CRD
// to the proto enum values understood by istio.io/api. An empty string keeps
// the zero value (PASSTHROUGH), which matches the proto default and the
// behaviour before the CRD-compat wrapper types were introduced.
var istioTLSModes = map[string]istiov1beta1.ServerTLSSettings_TLSmode{
	"":                 istiov1beta1.ServerTLSSettings_PASSTHROUGH,
	"PASSTHROUGH":      istiov1beta1.ServerTLSSettings_PASSTHROUGH,
	"SIMPLE":           istiov1beta1.ServerTLSSettings_SIMPLE,
	"MUTUAL":           istiov1beta1.ServerTLSSettings_MUTUAL,
	"AUTO_PASSTHROUGH": istiov1beta1.ServerTLSSettings_AUTO_PASSTHROUGH,
	"ISTIO_MUTUAL":     istiov1beta1.ServerTLSSettings_ISTIO_MUTUAL,
}

func istioTransformers(ctx context.Context, instance *servingv1beta1.KnativeServing) []mf.Transformer {
	logger := logging.FromContext(ctx)
	return []mf.Transformer{gatewayTransform(instance, logger)}
}

func gatewayTransform(instance *servingv1beta1.KnativeServing, log *zap.SugaredLogger) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		// Update the deployment with the new registry and tag
		if u.GetKind() == "Gateway" {
			if !strings.HasPrefix(u.GetAPIVersion(), "networking.istio.io/") {
				return nil
			}
			beta := true
			if strings.HasSuffix(u.GetAPIVersion(), "v1alpha3") {
				u.SetAPIVersion("networking.istio.io/v1beta1")
				beta = false
			}

			gateway := &istionetworkingv1beta.Gateway{}
			err := scheme.Scheme.Convert(u, gateway, nil)
			if err != nil {
				return err
			}

			if u.GetName() == "knative-ingress-gateway" {
				if err := updateIstioGateway(ingressGateway(instance), gateway, log); err != nil {
					return err
				}
			}
			// TODO: cluster-local-gateway was removed since v0.20 https://github.com/knative-extensions/net-istio/commit/058432d749435ef1fc61aa2b1fd048d0c75460ee
			// Remove it once operator stops v0.20 support.
			if u.GetName() == "cluster-local-gateway" || u.GetName() == "knative-local-gateway" {
				if err := updateIstioGateway(localGateway(instance), gateway, log); err != nil {
					return err
				}
			}

			if err := scheme.Scheme.Convert(gateway, u, nil); err != nil {
				return err
			}

			if !beta {
				u.SetAPIVersion("networking.istio.io/v1alpha3")
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

func updateIstioGateway(override *base.IstioGatewayOverride, gateway *istionetworkingv1beta.Gateway, log *zap.SugaredLogger) error {
	if override != nil && len(override.Selector) > 0 {
		log.Debugw("Updating Gateway", "name", gateway.GetName(), "gatewayOverrides", override)
		gateway.Spec.Selector = override.Selector
		log.Debugw("Finished conversion", "name", gateway.GetName())
	}

	if override != nil && len(override.Servers) > 0 {
		log.Debugw("Updating Gateway Servers", "name", gateway.GetName(), "gatewayOverrides", override)
		servers, err := toIstioServers(override.Servers)
		if err != nil {
			return fmt.Errorf("failed to convert servers override for gateway %q: %w", gateway.GetName(), err)
		}
		gateway.Spec.Servers = servers
		log.Debugw("Finished Servers Overrides", "name", gateway.GetName())
	}
	return nil
}

// toIstioServers converts the CRD-facing wrapper types in
// pkg/apis/operator/base to the proto-based types consumed by istio.io/api.
// A nil or empty input returns (nil, nil) so that callers can keep their
// existing emptiness checks.
func toIstioServers(in []base.IstioServer) ([]*istiov1beta1.Server, error) {
	if len(in) == 0 {
		return nil, nil
	}
	out := make([]*istiov1beta1.Server, 0, len(in))
	for i := range in {
		src := &in[i]
		dst := &istiov1beta1.Server{
			Bind:            src.Bind,
			Hosts:           append([]string(nil), src.Hosts...),
			Name:            src.Name,
			DefaultEndpoint: src.DefaultEndpoint,
		}
		if src.Port != nil {
			dst.Port = &istiov1beta1.Port{
				Number:     src.Port.Number,
				Protocol:   src.Port.Protocol,
				Name:       src.Port.Name,
				TargetPort: src.Port.TargetPort,
			}
		}
		if src.Tls != nil {
			mode, ok := istioTLSModes[src.Tls.Mode]
			if !ok {
				return nil, fmt.Errorf("unknown TLS mode %q: must be one of PASSTHROUGH, SIMPLE, MUTUAL, AUTO_PASSTHROUGH, ISTIO_MUTUAL", src.Tls.Mode)
			}
			dst.Tls = &istiov1beta1.ServerTLSSettings{
				HttpsRedirect:         src.Tls.HttpsRedirect,
				Mode:                  mode,
				ServerCertificate:     src.Tls.ServerCertificate,
				PrivateKey:            src.Tls.PrivateKey,
				CaCertificates:        src.Tls.CaCertificates,
				CredentialName:        src.Tls.CredentialName,
				SubjectAltNames:       append([]string(nil), src.Tls.SubjectAltNames...),
				VerifyCertificateSpki: append([]string(nil), src.Tls.VerifyCertificateSpki...),
				VerifyCertificateHash: append([]string(nil), src.Tls.VerifyCertificateHash...),
				MinProtocolVersion:    istiov1beta1.ServerTLSSettings_TLSProtocol(src.Tls.MinProtocolVersion),
				MaxProtocolVersion:    istiov1beta1.ServerTLSSettings_TLSProtocol(src.Tls.MaxProtocolVersion),
				CipherSuites:          append([]string(nil), src.Tls.CipherSuites...),
			}
		}
		out = append(out, dst)
	}
	return out, nil
}
