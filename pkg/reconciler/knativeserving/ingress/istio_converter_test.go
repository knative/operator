/*
Copyright 2026 The Knative Authors

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
	"reflect"
	"strings"
	"testing"

	istiov1beta1 "istio.io/api/networking/v1beta1"
	"knative.dev/operator/pkg/apis/operator/base"
)

func TestToIstioServers_NilInput(t *testing.T) {
	got, err := toIstioServers(nil)
	if err != nil {
		t.Fatalf("toIstioServers(nil) returned error: %v", err)
	}
	if got != nil {
		t.Fatalf("toIstioServers(nil) = %v, want nil", got)
	}
}

func TestToIstioServers_EmptyInput(t *testing.T) {
	got, err := toIstioServers([]base.IstioServer{})
	if err != nil {
		t.Fatalf("toIstioServers([]) returned error: %v", err)
	}
	if got != nil {
		t.Fatalf("toIstioServers([]) = %v, want nil", got)
	}
}

func TestToIstioServers_TLSModes(t *testing.T) {
	cases := []struct {
		mode string
		want istiov1beta1.ServerTLSSettings_TLSmode
	}{
		{"PASSTHROUGH", istiov1beta1.ServerTLSSettings_PASSTHROUGH},
		{"SIMPLE", istiov1beta1.ServerTLSSettings_SIMPLE},
		{"MUTUAL", istiov1beta1.ServerTLSSettings_MUTUAL},
		{"AUTO_PASSTHROUGH", istiov1beta1.ServerTLSSettings_AUTO_PASSTHROUGH},
		{"ISTIO_MUTUAL", istiov1beta1.ServerTLSSettings_ISTIO_MUTUAL},
	}

	// Confidence check: the proto enum still uses the historical 0..4 int32 layout.
	wantInts := []int32{0, 1, 2, 3, 4}
	for i, c := range cases {
		if int32(c.want) != wantInts[i] {
			t.Fatalf("proto enum %s moved from %d to %d; review converter",
				c.mode, wantInts[i], int32(c.want))
		}
	}

	for _, c := range cases {
		t.Run(c.mode, func(t *testing.T) {
			in := []base.IstioServer{{
				Tls: &base.IstioServerTLSSettings{Mode: c.mode},
			}}
			out, err := toIstioServers(in)
			if err != nil {
				t.Fatalf("toIstioServers returned error for mode %q: %v", c.mode, err)
			}
			if len(out) != 1 || out[0].Tls == nil {
				t.Fatalf("toIstioServers returned %v, want single server with non-nil Tls", out)
			}
			if out[0].Tls.Mode != c.want {
				t.Fatalf("Tls.Mode = %v (int32 %d), want %v (int32 %d)",
					out[0].Tls.Mode, int32(out[0].Tls.Mode), c.want, int32(c.want))
			}
		})
	}
}

func TestToIstioServers_EmptyModeDefaultsToPassthrough(t *testing.T) {
	in := []base.IstioServer{{
		Tls: &base.IstioServerTLSSettings{Mode: ""},
	}}
	out, err := toIstioServers(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 1 || out[0].Tls == nil {
		t.Fatalf("unexpected output: %v", out)
	}
	if out[0].Tls.Mode != istiov1beta1.ServerTLSSettings_PASSTHROUGH {
		t.Fatalf("empty mode = %v, want PASSTHROUGH (0)", out[0].Tls.Mode)
	}
	if int32(out[0].Tls.Mode) != 0 {
		t.Fatalf("empty mode int32 = %d, want 0", int32(out[0].Tls.Mode))
	}
}

func TestToIstioServers_UnknownModeReturnsError(t *testing.T) {
	in := []base.IstioServer{{
		Tls: &base.IstioServerTLSSettings{Mode: "BOGUS"},
	}}
	out, err := toIstioServers(in)
	if err == nil {
		t.Fatalf("expected error for unknown TLS mode, got nil (output=%v)", out)
	}
	if !strings.Contains(err.Error(), "BOGUS") {
		t.Fatalf("error %q does not mention bad mode value", err.Error())
	}
}

func TestToIstioServers_NilTlsDoesNotPanic(t *testing.T) {
	in := []base.IstioServer{{
		Hosts: []string{"example.com"},
		Port:  &base.IstioPort{Number: 80, Protocol: "HTTP", Name: "http"},
		Tls:   nil,
	}}
	out, err := toIstioServers(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("len(out) = %d, want 1", len(out))
	}
	if out[0].Tls != nil {
		t.Fatalf("Tls = %v, want nil", out[0].Tls)
	}
}

func TestToIstioServers_NilPortDoesNotPanic(t *testing.T) {
	in := []base.IstioServer{{
		Hosts: []string{"example.com"},
		Port:  nil,
		Tls:   &base.IstioServerTLSSettings{Mode: "SIMPLE"},
	}}
	out, err := toIstioServers(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("len(out) = %d, want 1", len(out))
	}
	if out[0].Port != nil {
		t.Fatalf("Port = %v, want nil", out[0].Port)
	}
	if out[0].Tls == nil || out[0].Tls.Mode != istiov1beta1.ServerTLSSettings_SIMPLE {
		t.Fatalf("Tls = %v, want non-nil with Mode=SIMPLE", out[0].Tls)
	}
}

// TestToIstioServers_FullFixture exercises every wrapper field on IstioServer
// and asserts 1:1 mapping onto the proto-derived istio.io/api types.
func TestToIstioServers_FullFixture(t *testing.T) {
	in := []base.IstioServer{
		{
			Bind:            "0.0.0.0",
			Hosts:           []string{"example.com", "foo.example.com"},
			Name:            "https-ingress",
			DefaultEndpoint: "127.0.0.1:8080",
			Port: &base.IstioPort{
				Number:     443,
				Protocol:   "HTTPS",
				Name:       "https",
				TargetPort: 8443,
			},
			Tls: &base.IstioServerTLSSettings{
				HttpsRedirect:         true,
				Mode:                  "MUTUAL",
				ServerCertificate:     "/etc/certs/server.crt",
				PrivateKey:            "/etc/certs/server.key",
				CaCertificates:        "/etc/certs/ca.crt",
				CredentialName:        "my-secret",
				SubjectAltNames:       []string{"spiffe://cluster.local/ns/default/sa/foo"},
				VerifyCertificateSpki: []string{"abc123"},
				VerifyCertificateHash: []string{"def456", "ffffff"},
				MinProtocolVersion:    2,
				MaxProtocolVersion:    3,
				CipherSuites:          []string{"ECDHE-RSA-AES256-GCM-SHA384"},
			},
		},
		{
			// Second server: HTTP-only, with no TLS to confirm coexistence.
			Bind:  "",
			Hosts: []string{"plain.example.com"},
			Name:  "http-ingress",
			Port: &base.IstioPort{
				Number:   80,
				Protocol: "HTTP",
				Name:     "http",
			},
		},
	}

	out, err := toIstioServers(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("len(out) = %d, want 2", len(out))
	}

	// Server 0
	s0 := out[0]
	if s0.Bind != "0.0.0.0" {
		t.Errorf("s0.Bind = %q, want 0.0.0.0", s0.Bind)
	}
	if !reflect.DeepEqual(s0.Hosts, []string{"example.com", "foo.example.com"}) {
		t.Errorf("s0.Hosts = %v, want [example.com foo.example.com]", s0.Hosts)
	}
	if s0.Name != "https-ingress" {
		t.Errorf("s0.Name = %q, want https-ingress", s0.Name)
	}
	if s0.DefaultEndpoint != "127.0.0.1:8080" {
		t.Errorf("s0.DefaultEndpoint = %q, want 127.0.0.1:8080", s0.DefaultEndpoint)
	}
	if s0.Port == nil {
		t.Fatalf("s0.Port = nil")
	}
	if s0.Port.Number != 443 || s0.Port.Protocol != "HTTPS" ||
		s0.Port.Name != "https" || s0.Port.TargetPort != 8443 { //nolint:staticcheck // TargetPort is deprecated but still populated by the converter.
		t.Errorf("s0.Port = %+v, want {Number:443 Protocol:HTTPS Name:https TargetPort:8443}", s0.Port)
	}
	if s0.Tls == nil {
		t.Fatalf("s0.Tls = nil")
	}
	wantTLS := &istiov1beta1.ServerTLSSettings{
		HttpsRedirect:         true,
		Mode:                  istiov1beta1.ServerTLSSettings_MUTUAL,
		ServerCertificate:     "/etc/certs/server.crt",
		PrivateKey:            "/etc/certs/server.key",
		CaCertificates:        "/etc/certs/ca.crt",
		CredentialName:        "my-secret",
		SubjectAltNames:       []string{"spiffe://cluster.local/ns/default/sa/foo"},
		VerifyCertificateSpki: []string{"abc123"},
		VerifyCertificateHash: []string{"def456", "ffffff"},
		MinProtocolVersion:    istiov1beta1.ServerTLSSettings_TLSProtocol(2),
		MaxProtocolVersion:    istiov1beta1.ServerTLSSettings_TLSProtocol(3),
		CipherSuites:          []string{"ECDHE-RSA-AES256-GCM-SHA384"},
	}
	// Field-by-field check (can't use proto.Equal without the proto pkg and
	// reflect.DeepEqual would trip over internal proto state, so compare
	// each exported field explicitly).
	if s0.Tls.HttpsRedirect != wantTLS.HttpsRedirect {
		t.Errorf("HttpsRedirect = %v, want %v", s0.Tls.HttpsRedirect, wantTLS.HttpsRedirect)
	}
	if s0.Tls.Mode != wantTLS.Mode {
		t.Errorf("Mode = %v, want %v", s0.Tls.Mode, wantTLS.Mode)
	}
	if s0.Tls.ServerCertificate != wantTLS.ServerCertificate {
		t.Errorf("ServerCertificate = %q, want %q", s0.Tls.ServerCertificate, wantTLS.ServerCertificate)
	}
	if s0.Tls.PrivateKey != wantTLS.PrivateKey {
		t.Errorf("PrivateKey = %q, want %q", s0.Tls.PrivateKey, wantTLS.PrivateKey)
	}
	if s0.Tls.CaCertificates != wantTLS.CaCertificates {
		t.Errorf("CaCertificates = %q, want %q", s0.Tls.CaCertificates, wantTLS.CaCertificates)
	}
	if s0.Tls.CredentialName != wantTLS.CredentialName {
		t.Errorf("CredentialName = %q, want %q", s0.Tls.CredentialName, wantTLS.CredentialName)
	}
	if !reflect.DeepEqual(s0.Tls.SubjectAltNames, wantTLS.SubjectAltNames) {
		t.Errorf("SubjectAltNames = %v, want %v", s0.Tls.SubjectAltNames, wantTLS.SubjectAltNames)
	}
	if !reflect.DeepEqual(s0.Tls.VerifyCertificateSpki, wantTLS.VerifyCertificateSpki) {
		t.Errorf("VerifyCertificateSpki = %v, want %v", s0.Tls.VerifyCertificateSpki, wantTLS.VerifyCertificateSpki)
	}
	if !reflect.DeepEqual(s0.Tls.VerifyCertificateHash, wantTLS.VerifyCertificateHash) {
		t.Errorf("VerifyCertificateHash = %v, want %v", s0.Tls.VerifyCertificateHash, wantTLS.VerifyCertificateHash)
	}
	if s0.Tls.MinProtocolVersion != wantTLS.MinProtocolVersion {
		t.Errorf("MinProtocolVersion = %v, want %v", s0.Tls.MinProtocolVersion, wantTLS.MinProtocolVersion)
	}
	if s0.Tls.MaxProtocolVersion != wantTLS.MaxProtocolVersion {
		t.Errorf("MaxProtocolVersion = %v, want %v", s0.Tls.MaxProtocolVersion, wantTLS.MaxProtocolVersion)
	}
	if !reflect.DeepEqual(s0.Tls.CipherSuites, wantTLS.CipherSuites) {
		t.Errorf("CipherSuites = %v, want %v", s0.Tls.CipherSuites, wantTLS.CipherSuites)
	}

	// Server 1 — no TLS, only HTTP port.
	s1 := out[1]
	if s1.Bind != "" {
		t.Errorf("s1.Bind = %q, want empty", s1.Bind)
	}
	if !reflect.DeepEqual(s1.Hosts, []string{"plain.example.com"}) {
		t.Errorf("s1.Hosts = %v", s1.Hosts)
	}
	if s1.Name != "http-ingress" {
		t.Errorf("s1.Name = %q, want http-ingress", s1.Name)
	}
	if s1.Port == nil || s1.Port.Number != 80 || s1.Port.Protocol != "HTTP" || s1.Port.Name != "http" {
		t.Errorf("s1.Port = %+v, want {Number:80 Protocol:HTTP Name:http}", s1.Port)
	}
	if s1.Tls != nil {
		t.Errorf("s1.Tls = %v, want nil", s1.Tls)
	}
}

// TestToIstioServers_HostsAreCopied confirms the converter does not alias the
// input slice — mutating the output must not affect the input.
func TestToIstioServers_HostsAreCopied(t *testing.T) {
	in := []base.IstioServer{{
		Hosts: []string{"a", "b"},
		Tls: &base.IstioServerTLSSettings{
			Mode:            "SIMPLE",
			SubjectAltNames: []string{"x", "y"},
		},
	}}
	out, err := toIstioServers(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out[0].Hosts[0] = "mutated"
	if in[0].Hosts[0] != "a" {
		t.Errorf("input Hosts aliased to output: got %q", in[0].Hosts[0])
	}
	out[0].Tls.SubjectAltNames[0] = "mutated"
	if in[0].Tls.SubjectAltNames[0] != "x" {
		t.Errorf("input SubjectAltNames aliased to output: got %q", in[0].Tls.SubjectAltNames[0])
	}
}
