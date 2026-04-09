/*
Copyright 2025 The Knative Authors

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
	"context"
	"strings"
	"testing"

	mf "github.com/manifestival/manifestival"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"

	"knative.dev/operator/pkg/apis/operator/base"
	"knative.dev/operator/pkg/apis/operator/v1beta1"
)

func TestResolveTargetCluster_NilRef(t *testing.T) {
	instance := &v1beta1.KnativeServing{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
	}
	instance.Status.InitializeConditions()

	manifest, err := mf.ManifestFrom(mf.Slice{})
	if err != nil {
		t.Fatalf("Failed to create manifest: %v", err)
	}

	origClient := manifest.Client

	var state ReconcileState
	stage := ResolveTargetCluster(nil, &state)
	if err := stage(context.Background(), &manifest, instance); err != nil {
		t.Fatalf("ResolveTargetCluster() = %v, want nil", err)
	}

	if manifest.Client != origClient {
		t.Fatal("manifest.Client changed unexpectedly")
	}

	if state.AnchorOwner != nil {
		t.Fatal("state.AnchorOwner is non-nil, want nil")
	}

	if state.IsRemote() {
		t.Fatal("state.IsRemote() = true, want false")
	}

	cond := instance.Status.GetCondition(base.TargetClusterResolved)
	if cond == nil || cond.Status != corev1.ConditionTrue {
		t.Fatalf("TargetClusterResolved = %v, want True", cond)
	}
}

func TestEnsureAnchorConfigMap_Create(t *testing.T) {
	kubeClient := fake.NewSimpleClientset()
	instance := &v1beta1.KnativeServing{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-ns",
			Name:      "test",
		},
	}

	ctx := context.Background()
	anchor, err := EnsureAnchorConfigMap(ctx, kubeClient, instance)
	if err != nil {
		t.Fatalf("EnsureAnchorConfigMap() error: %v", err)
	}

	expectedName := "knativeserving-test-root-owner"
	if anchor.Name != expectedName {
		t.Fatalf("anchor.Name = %q, want %q", anchor.Name, expectedName)
	}
	if anchor.Namespace != "test-ns" {
		t.Fatalf("anchor.Namespace = %q, want %q", anchor.Namespace, "test-ns")
	}

	if got := anchor.Labels["app.kubernetes.io/managed-by"]; got != "knative-operator" {
		t.Fatalf("label managed-by = %q, want %q", got, "knative-operator")
	}
	if got := anchor.Labels["operator.knative.dev/cr-name"]; got != "test" {
		t.Fatalf("label cr-name = %q, want %q", got, "test")
	}

	if got := anchor.Annotations["operator.knative.dev/anchor"]; got != "true" {
		t.Fatalf("annotation anchor = %q, want %q", got, "true")
	}
	if _, ok := anchor.Annotations["operator.knative.dev/warning"]; ok {
		t.Fatal("unexpected warning annotation on newly created anchor")
	}

	ns, err := kubeClient.CoreV1().Namespaces().Get(ctx, "test-ns", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Get namespace: %v", err)
	}
	if got := ns.Labels["app.kubernetes.io/managed-by"]; got != "knative-operator" {
		t.Fatalf("namespace label managed-by = %q, want %q", got, "knative-operator")
	}
}

func TestEnsureAnchorConfigMap_AlreadyExists(t *testing.T) {
	existingAnchor := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "knativeserving-test-root-owner",
			Namespace: "test-ns",
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "knative-operator",
				"operator.knative.dev/cr-name": "test",
			},
			Annotations: map[string]string{
				"operator.knative.dev/anchor": "true",
			},
		},
	}
	existingNS := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "test-ns"},
	}
	kubeClient := fake.NewSimpleClientset(existingNS, existingAnchor)

	instance := &v1beta1.KnativeServing{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-ns",
			Name:      "test",
		},
	}

	ctx := context.Background()
	anchor, err := EnsureAnchorConfigMap(ctx, kubeClient, instance)
	if err != nil {
		t.Fatalf("EnsureAnchorConfigMap() error: %v", err)
	}

	if anchor.Name != "knativeserving-test-root-owner" {
		t.Fatalf("anchor.Name = %q, want %q", anchor.Name, "knativeserving-test-root-owner")
	}
	if anchor.Namespace != "test-ns" {
		t.Fatalf("anchor.Namespace = %q, want %q", anchor.Namespace, "test-ns")
	}
}

func TestEnsureAnchorConfigMap_AdditiveMerge(t *testing.T) {
	instance := &v1beta1.KnativeServing{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "knative-serving",
			Namespace: "knative-serving",
		},
	}
	oldAnchor := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      AnchorName(instance),
			Namespace: "knative-serving",
			Labels: map[string]string{
				"old-label": "old-value",
			},
			Annotations: map[string]string{
				"old-annotation": "old-value",
			},
		},
	}
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "knative-serving"},
	}
	kubeClient := fake.NewSimpleClientset(oldAnchor, ns)

	anchor, err := EnsureAnchorConfigMap(context.Background(), kubeClient, instance)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := anchor.Labels["app.kubernetes.io/managed-by"]; got != "knative-operator" {
		t.Errorf("managed-by label = %q, want %q", got, "knative-operator")
	}
	if got := anchor.Labels["operator.knative.dev/cr-name"]; got != "knative-serving" {
		t.Errorf("cr-name label = %q, want %q", got, "knative-serving")
	}
	if got := anchor.Labels["old-label"]; got != "old-value" {
		t.Errorf("old-label not preserved, got labels: %v", anchor.Labels)
	}

	if got := anchor.Annotations["operator.knative.dev/anchor"]; got != "true" {
		t.Errorf("anchor annotation = %q, want %q", got, "true")
	}
	if got := anchor.Annotations["old-annotation"]; got != "old-value" {
		t.Errorf("old-annotation not preserved, got annotations: %v", anchor.Annotations)
	}
}

func TestEnsureAnchorConfigMap_NameTooLong(t *testing.T) {
	longName := strings.Repeat("a", 250)
	instance := &v1beta1.KnativeServing{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-ns",
			Name:      longName,
		},
	}

	kubeClient := fake.NewSimpleClientset()
	_, err := EnsureAnchorConfigMap(context.Background(), kubeClient, instance)
	if err == nil {
		t.Fatal("EnsureAnchorConfigMap() = nil, want error")
	}
	if !strings.Contains(err.Error(), "exceeds maximum length") {
		t.Fatalf("error = %v, want substring %q", err, "exceeds maximum length")
	}
}

func TestDeleteAnchorConfigMap_Success(t *testing.T) {
	existingAnchor := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "knativeserving-test-root-owner",
			Namespace: "test-ns",
		},
	}
	kubeClient := fake.NewSimpleClientset(existingAnchor)

	instance := &v1beta1.KnativeServing{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-ns",
			Name:      "test",
		},
	}

	ctx := context.Background()
	if err := DeleteAnchorConfigMap(ctx, kubeClient, instance); err != nil {
		t.Fatalf("DeleteAnchorConfigMap() error: %v", err)
	}

	_, err := kubeClient.CoreV1().ConfigMaps("test-ns").Get(ctx, "knativeserving-test-root-owner", metav1.GetOptions{})
	if err == nil {
		t.Fatal("anchor ConfigMap still exists after deletion")
	}
}

func TestConfigEqual(t *testing.T) {
	cfg := &rest.Config{
		Host:            "https://example.com",
		BearerToken:     "token",
		BearerTokenFile: "/path/to/token",
		Username:        "user",
		Password:        "pass",
	}
	same := &rest.Config{
		Host:            "https://example.com",
		BearerToken:     "token",
		BearerTokenFile: "/path/to/token",
		Username:        "user",
		Password:        "pass",
	}
	if !configEqual(cfg, same) {
		t.Fatal("configEqual() = false, want true")
	}

	if configEqual(cfg, &rest.Config{Host: "https://other.com", BearerToken: "token"}) {
		t.Fatal("configEqual(different Host) = true, want false")
	}

	if configEqual(cfg, &rest.Config{Host: "https://example.com", BearerToken: "other-token"}) {
		t.Fatal("configEqual(different BearerToken) = true, want false")
	}

	if configEqual(cfg, &rest.Config{
		Host:            "https://example.com",
		TLSClientConfig: rest.TLSClientConfig{CertFile: "/path/to/cert"},
	}) {
		t.Fatal("configEqual(different TLSClientConfig) = true, want false")
	}
}

func TestSameClusterProfile(t *testing.T) {
	tests := []struct {
		name string
		a, b *base.ClusterProfileReference
		want bool
	}{
		{
			name: "both nil",
			a:    nil,
			b:    nil,
			want: true,
		},
		{
			name: "a nil",
			a:    nil,
			b:    &base.ClusterProfileReference{Namespace: "ns", Name: "name"},
			want: false,
		},
		{
			name: "same",
			a:    &base.ClusterProfileReference{Namespace: "ns", Name: "name"},
			b:    &base.ClusterProfileReference{Namespace: "ns", Name: "name"},
			want: true,
		},
		{
			name: "different",
			a:    &base.ClusterProfileReference{Namespace: "ns", Name: "name1"},
			b:    &base.ClusterProfileReference{Namespace: "ns", Name: "name2"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SameClusterProfile(tt.a, tt.b)
			if got != tt.want {
				t.Fatalf("SameClusterProfile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShouldFinalizeClusterScoped(t *testing.T) {
	ref := &base.ClusterProfileReference{Namespace: "fleet", Name: "spoke1"}

	tests := []struct {
		name       string
		components []base.KComponent
		original   base.KComponent
		want       bool
	}{
		{
			name:       "no other components",
			components: []base.KComponent{},
			original: &v1beta1.KnativeServing{
				ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: "ks"},
			},
			want: true,
		},
		{
			name: "another alive component with same cluster profile",
			components: []base.KComponent{
				&v1beta1.KnativeServing{
					ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: "ks-other"},
					Spec: v1beta1.KnativeServingSpec{
						CommonSpec: base.CommonSpec{ClusterProfileRef: ref},
					},
				},
			},
			original: &v1beta1.KnativeServing{
				ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: "ks"},
				Spec: v1beta1.KnativeServingSpec{
					CommonSpec: base.CommonSpec{ClusterProfileRef: ref},
				},
			},
			want: false,
		},
		{
			name: "another alive component with different cluster profile",
			components: []base.KComponent{
				&v1beta1.KnativeServing{
					ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: "ks-other"},
					Spec: v1beta1.KnativeServingSpec{
						CommonSpec: base.CommonSpec{ClusterProfileRef: &base.ClusterProfileReference{
							Namespace: "fleet", Name: "spoke2",
						}},
					},
				},
			},
			original: &v1beta1.KnativeServing{
				ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: "ks"},
				Spec: v1beta1.KnativeServingSpec{
					CommonSpec: base.CommonSpec{ClusterProfileRef: ref},
				},
			},
			want: true,
		},
		{
			name: "both local (nil refs), another alive",
			components: []base.KComponent{
				&v1beta1.KnativeServing{
					ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: "ks-other"},
				},
			},
			original: &v1beta1.KnativeServing{
				ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: "ks"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldFinalizeClusterScoped(tt.components, tt.original)
			if got != tt.want {
				t.Fatalf("ShouldFinalizeClusterScoped() = %v, want %v", got, tt.want)
			}
		})
	}
}
