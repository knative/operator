/*
Copyright 2022 The Knative Authors

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

package base

import (
	v1 "k8s.io/api/core/v1"
)

// IstioIngressConfiguration specifies options for the istio ingresses.
type IstioIngressConfiguration struct {
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// KnativeIngressGateway overrides the knative-ingress-gateway.
	// +optional
	KnativeIngressGateway *IstioGatewayOverride `json:"knative-ingress-gateway,omitempty"`

	// KnativeLocalGateway overrides the knative-local-gateway.
	// +optional
	KnativeLocalGateway *IstioGatewayOverride `json:"knative-local-gateway,omitempty"`
}

// KourierIngressConfiguration specifies whether to enable the kourier ingresses.
type KourierIngressConfiguration struct {
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// ServiceType specifies the service type for kourier gateway.
	ServiceType v1.ServiceType `json:"service-type,omitempty"`

	// ServiceLoadBalancerIP specifies the service load balancer IP.
	ServiceLoadBalancerIP string `json:"service-load-balancer-ip,omitempty"`

	// HTTPPort specifies the port used in case of ServiceType = "NodePort" for http traffic
	HTTPPort int32 `json:"http-port,omitempty"`

	// HTTPSPort specifies the port used in case of ServiceType = "NodePort" for https (encrypted) traffic
	HTTPSPort int32 `json:"https-port,omitempty"`

	// BootstrapConfigmapName specifies the ConfigMap name which contains envoy bootstrap.
	BootstrapConfigmapName string `json:"bootstrap-configmap,omitempty"`
}

// ContourIngressConfiguration specifies whether to enable the contour ingresses.
type ContourIngressConfiguration struct {
	// +optional
	Enabled bool `json:"enabled,omitempty"`
}

// GatewayAPIIngressConfiguration specifies whether to enable the gateway-api ingresses.
type GatewayAPIIngressConfiguration struct {
	// +optional
	Enabled bool `json:"enabled,omitempty"`
}

// IstioGatewayOverride override the knative-ingress-gateway and knative-local-gateway(cluster-local-gateway)
type IstioGatewayOverride struct {
	// A map of values to replace the "selector" values in the knative-ingress-gateway and knative-local-gateway(cluster-local-gateway)
	Selector map[string]string `json:"selector,omitempty"`

	// Servers is a list of server specifications applied to the Istio Gateway.
	// +optional
	Servers []IstioServer `json:"servers,omitempty"`
}

// IstioServer describes the properties of the proxy on a given load balancer port.
// See https://istio.io/latest/docs/reference/config/networking/gateway/#Server.
type IstioServer struct {
	// Port on which the proxy should listen for incoming connections.
	// +optional
	Port *IstioPort `json:"port,omitempty"`

	// Bind is the IP or the Unix domain socket to which the listener should
	// be bound to. Format: `x.x.x.x` or `unix:///path/to/uds` or
	// `unix://@foobar` (Linux abstract namespace). When using Unix domain
	// sockets, the port number should be 0.
	// +optional
	Bind string `json:"bind,omitempty"`

	// Hosts is one or more hosts exposed by this gateway. A host is specified
	// as a `dnsName` with an optional `namespace/` prefix.
	// +optional
	Hosts []string `json:"hosts,omitempty"`

	// Name is an optional name of the server. When set it must be unique
	// across all servers on a single Gateway.
	// +optional
	Name string `json:"name,omitempty"`

	// Tls configures TLS settings for the server.
	// +optional
	Tls *IstioServerTLSSettings `json:"tls,omitempty"`

	// DefaultEndpoint is the loopback IP endpoint or Unix domain socket to
	// which traffic should be forwarded to by default.
	// +optional
	DefaultEndpoint string `json:"defaultEndpoint,omitempty"`
}

// IstioPort describes the port on which a Gateway server should listen.
// See https://istio.io/latest/docs/reference/config/networking/gateway/#Port.
type IstioPort struct {
	// Number is a valid non-negative integer port number.
	// +optional
	Number uint32 `json:"number,omitempty"`

	// Protocol exposed on the port. MUST BE one of
	// HTTP|HTTPS|GRPC|HTTP2|MONGO|TCP|TLS.
	// +optional
	Protocol string `json:"protocol,omitempty"`

	// Name is a label assigned to the port.
	// +optional
	Name string `json:"name,omitempty"`

	// TargetPort is the target port on the workload.
	// The snake_case JSON tag is preserved for backward compatibility with existing KnativeServing CRs.
	// +optional
	TargetPort uint32 `json:"target_port,omitempty"`
}

// IstioServerTLSSettings configures TLS for an Istio Gateway server.
// See https://istio.io/latest/docs/reference/config/networking/gateway/#ServerTLSSettings.
type IstioServerTLSSettings struct {
	// HttpsRedirect, if set to true, causes the load balancer to send a 301
	// redirect to HTTPS for all HTTP requests. Should only be used on HTTP
	// listeners and is mutually exclusive with all other TLS options.
	// +optional
	HttpsRedirect bool `json:"httpsRedirect,omitempty"`

	// Mode indicates whether connections should be secured by TLS.
	// +optional
	// +kubebuilder:validation:Enum=PASSTHROUGH;SIMPLE;MUTUAL;AUTO_PASSTHROUGH;ISTIO_MUTUAL
	Mode string `json:"mode,omitempty"`

	// ServerCertificate is the path to the file holding the server-side TLS
	// certificate to use.
	// +optional
	ServerCertificate string `json:"serverCertificate,omitempty"`

	// PrivateKey is the path to the file holding the server's private key.
	// +optional
	PrivateKey string `json:"privateKey,omitempty"`

	// CaCertificates is the path to a file containing certificate authority
	// certificates to use in verifying a presented client side certificate.
	// +optional
	CaCertificates string `json:"caCertificates,omitempty"`

	// CredentialName is the name of the secret holding the server-side TLS
	// certificate to use.
	// +optional
	CredentialName string `json:"credentialName,omitempty"`

	// SubjectAltNames is a list of alternate names to verify the subject
	// identity in the certificate presented by the client.
	// +optional
	SubjectAltNames []string `json:"subjectAltNames,omitempty"`

	// VerifyCertificateSpki is an optional list of base64-encoded SHA-256
	// hashes of the SPKIs of authorized client certificates.
	// +optional
	VerifyCertificateSpki []string `json:"verifyCertificateSpki,omitempty"`

	// VerifyCertificateHash is an optional list of hex-encoded SHA-256 hashes
	// of the authorized client certificates.
	// +optional
	VerifyCertificateHash []string `json:"verifyCertificateHash,omitempty"`

	// MinProtocolVersion is the minimum TLS protocol version.
	// +optional
	MinProtocolVersion int32 `json:"minProtocolVersion,omitempty"`

	// MaxProtocolVersion is the maximum TLS protocol version.
	// +optional
	MaxProtocolVersion int32 `json:"maxProtocolVersion,omitempty"`

	// CipherSuites is an optional list of cipher suites to be used when
	// negotiating TLS.
	// +optional
	CipherSuites []string `json:"cipherSuites,omitempty"`
}
