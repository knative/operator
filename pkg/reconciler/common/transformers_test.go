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
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	mf "github.com/manifestival/manifestival"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"knative.dev/operator/pkg/apis/operator/v1beta1"
	"knative.dev/pkg/ptr"
)

func TestCommonTransformers(t *testing.T) {
	component := &v1beta1.KnativeEventing{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-ns",
			Name:      "test-name",
		},
	}
	in := []unstructured.Unstructured{*NamespacedResource("test/v1", "TestCR", "another-ns", "test-resource")}
	manifest, err := mf.ManifestFrom(mf.Slice(in))
	if err != nil {
		t.Fatalf("Failed to generate manifest: %v", err)
	}
	if err := Transform(context.Background(), &manifest, component, InjectOwner(component, nil)); err != nil {
		t.Fatalf("Failed to transform manifest: %v", err)
	}
	resource := &manifest.Resources()[0]

	// Verify namespace is carried over.
	if got, want := resource.GetNamespace(), component.GetNamespace(); got != want {
		t.Fatalf("GetNamespace() = %s, want %s", got, want)
	}

	// Transform with a platform extension
	ext := TestExtension("fubar")
	if err := Transform(context.Background(), &manifest, component, ext.Transformers(component)...); err != nil {
		t.Fatalf("Failed to transform manifest: %v", err)
	}
	resource = &manifest.Resources()[0]

	// Verify namespace is transformed
	if got, want := resource.GetNamespace(), string(ext); got != want {
		t.Fatalf("GetNamespace() = %s, want %s", got, want)
	}

	// Verify OwnerReference is set.
	if len(resource.GetOwnerReferences()) == 0 {
		t.Fatalf("len(GetOwnerReferences()) = 0, expected at least 1")
	}
	ownerRef := resource.GetOwnerReferences()[0]

	apiVersion, kind := component.GroupVersionKind().ToAPIVersionAndKind()
	wantOwnerRef := metav1.OwnerReference{
		APIVersion:         apiVersion,
		Kind:               kind,
		Name:               component.GetName(),
		Controller:         ptr.Bool(true),
		BlockOwnerDeletion: ptr.Bool(true),
	}

	if !cmp.Equal(ownerRef, wantOwnerRef) {
		t.Fatalf("Unexpected ownerRef: %s", cmp.Diff(ownerRef, wantOwnerRef))
	}
}

func TestInjectOwner_UsesAnchorWhenSet(t *testing.T) {
	component := &v1beta1.KnativeEventing{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-ns",
			Name:      "test-name",
		},
	}

	anchor := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "knativeeventing-test-name-root-owner",
			Namespace: "test-ns",
			UID:       types.UID("anchor-uid-123"),
		},
	}
	anchor.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("ConfigMap"))

	in := []unstructured.Unstructured{*NamespacedResource("test/v1", "TestCR", "some-ns", "test-resource")}
	manifest, err := mf.ManifestFrom(mf.Slice(in))
	if err != nil {
		t.Fatalf("Failed to generate manifest: %v", err)
	}

	transformer := InjectOwner(component, anchor)
	m, err := manifest.Transform(transformer)
	if err != nil {
		t.Fatalf("Failed to transform manifest: %v", err)
	}

	resource := &m.Resources()[0]
	if len(resource.GetOwnerReferences()) == 0 {
		t.Fatal("len(GetOwnerReferences()) = 0, expected at least 1")
	}

	ownerRef := resource.GetOwnerReferences()[0]
	if ownerRef.Name != anchor.Name {
		t.Fatalf("ownerRef.Name = %q, want %q (anchor name)", ownerRef.Name, anchor.Name)
	}
	if ownerRef.Kind != "ConfigMap" {
		t.Fatalf("ownerRef.Kind = %q, want %q", ownerRef.Kind, "ConfigMap")
	}
	if ownerRef.UID != anchor.UID {
		t.Fatalf("ownerRef.UID = %q, want %q", ownerRef.UID, anchor.UID)
	}
}

func TestInjectOwner_UsesCRWhenAnchorNil(t *testing.T) {
	component := &v1beta1.KnativeEventing{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-ns",
			Name:      "test-name",
		},
	}
	in := []unstructured.Unstructured{*NamespacedResource("test/v1", "TestCR", "some-ns", "test-resource")}
	manifest, err := mf.ManifestFrom(mf.Slice(in))
	if err != nil {
		t.Fatalf("Failed to generate manifest: %v", err)
	}

	transformer := InjectOwner(component, nil)
	m, err := manifest.Transform(transformer)
	if err != nil {
		t.Fatalf("Failed to transform manifest: %v", err)
	}

	resource := &m.Resources()[0]
	if len(resource.GetOwnerReferences()) == 0 {
		t.Fatal("len(GetOwnerReferences()) = 0, expected at least 1")
	}

	ownerRef := resource.GetOwnerReferences()[0]
	apiVersion, kind := component.GroupVersionKind().ToAPIVersionAndKind()
	wantOwnerRef := metav1.OwnerReference{
		APIVersion:         apiVersion,
		Kind:               kind,
		Name:               component.GetName(),
		Controller:         ptr.Bool(true),
		BlockOwnerDeletion: ptr.Bool(true),
	}

	if !cmp.Equal(ownerRef, wantOwnerRef) {
		t.Fatalf("Unexpected ownerRef: %s", cmp.Diff(ownerRef, wantOwnerRef))
	}
}

func TestInjectOwner_SkipsClusterScoped(t *testing.T) {
	component := &v1beta1.KnativeEventing{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-ns",
			Name:      "test-name",
		},
	}

	tests := []struct {
		name        string
		anchorOwner mf.Owner
	}{
		{
			name:        "without anchor",
			anchorOwner: nil,
		},
		{
			name: "with anchor",
			anchorOwner: func() mf.Owner {
				anchor := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "anchor",
						Namespace: "test-ns",
						UID:       types.UID("anchor-uid"),
					},
				}
				anchor.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("ConfigMap"))
				return anchor
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := []unstructured.Unstructured{*ClusterScopedResource("test/v1", "TestCR", "test-resource")}
			manifest, err := mf.ManifestFrom(mf.Slice(in))
			if err != nil {
				t.Fatalf("Failed to generate manifest: %v", err)
			}

			transformer := InjectOwner(component, tt.anchorOwner)
			m, err := manifest.Transform(transformer)
			if err != nil {
				t.Fatalf("Failed to transform manifest: %v", err)
			}

			resource := &m.Resources()[0]
			if got := len(resource.GetOwnerReferences()); got != 0 {
				t.Fatalf("len(GetOwnerReferences()) = %d, want 0",
					got)
			}
		})
	}
}

func TestInjectNamespace(t *testing.T) {
	component := &v1beta1.KnativeEventing{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-ns",
			Name:      "test-name",
		},
	}
	in := []unstructured.Unstructured{*NamespacedResource("test/v1", "TestCR", "another-ns", "test-resource")}
	manifest, err := mf.ManifestFrom(mf.Slice(in))
	if err != nil {
		t.Fatalf("Failed to generate manifest: %v", err)
	}
	if err := InjectNamespace(&manifest, component); err != nil {
		t.Fatalf("Failed to transform manifest: %v", err)
	}
	resource := &manifest.Resources()[0]

	// Verify namespace is carried over.
	if got, want := resource.GetNamespace(), component.GetNamespace(); got != want {
		t.Fatalf("GetNamespace() = %s, want %s", got, want)
	}
}
