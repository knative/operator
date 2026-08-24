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

package knativeserving

import (
	"context"
	"testing"

	mf "github.com/manifestival/manifestival"
	mffake "github.com/manifestival/manifestival/fake"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"knative.dev/operator/pkg/apis/operator/base"
	"knative.dev/operator/pkg/apis/operator/v1beta1"
	"knative.dev/operator/pkg/reconciler/common"
)

type extensionForPatchTest struct {
	manifest mf.Manifest
}

func (e extensionForPatchTest) Manifests(base.KComponent) ([]mf.Manifest, error) {
	return []mf.Manifest{e.manifest}, nil
}

func (extensionForPatchTest) Transformers(base.KComponent) []mf.Transformer {
	return []mf.Transformer{mf.InjectNamespace("extension-system")}
}

func (extensionForPatchTest) Reconcile(context.Context, base.KComponent) error {
	return nil
}

func (extensionForPatchTest) Finalize(context.Context, base.KComponent) error {
	return nil
}

func TestDeletedExtensionResourceIsPruned(t *testing.T) {
	common.ClearCache()
	client := mffake.New()
	baseManifest, err := mf.ManifestFrom(
		mf.Slice([]unstructured.Unstructured{}),
		mf.UseClient(client),
	)
	if err != nil {
		t.Fatalf("ManifestFrom() = %v", err)
	}
	extensionResource := common.NamespacedResource("v1", "ConfigMap", "upstream", "extension-config")
	extensionManifest, err := mf.ManifestFrom(mf.Slice([]unstructured.Unstructured{*extensionResource}))
	if err != nil {
		t.Fatalf("ManifestFrom() for extension = %v", err)
	}
	reconciler := &Reconciler{
		manifest:  baseManifest,
		extension: extensionForPatchTest{manifest: extensionManifest},
	}
	instance := &v1beta1.KnativeServing{
		ObjectMeta: metav1.ObjectMeta{Name: "knative-serving", Namespace: "knative-serving"},
		Spec: v1beta1.KnativeServingSpec{CommonSpec: base.CommonSpec{
			Patches: []base.ResourcePatch{{
				Target: base.PatchTarget{
					APIVersion: "v1",
					Kind:       "ConfigMap",
					Name:       "extension-config",
					Namespace:  "extension-system",
				},
				Patch: base.PatchSpec{Type: base.StrategicMergePatchType, Content: `$patch: delete`},
			}},
		}},
		Status: v1beta1.KnativeServingStatus{
			Manifests: []string{"../common/testdata/kodata/additional-manifests/additional-sa.yaml"},
		},
	}

	installed, err := reconciler.installed(context.Background(), instance)
	if err != nil {
		t.Fatalf("installed() = %v", err)
	}
	if err := installed.Apply(); err != nil {
		t.Fatalf("installed.Apply() = %v", err)
	}
	desired := installed.Append()
	if err := common.ApplyResourcePatches(&desired, instance.GetSpec().GetPatches()); err != nil {
		t.Fatalf("ApplyResourcePatches() = %v", err)
	}
	deleteObsolete := common.DeleteObsoleteResources(context.Background(), instance, func(context.Context, base.KComponent) (*mf.Manifest, error) {
		return installed, nil
	})
	if err := deleteObsolete(context.Background(), &desired, instance); err != nil {
		t.Fatalf("DeleteObsoleteResources() = %v", err)
	}

	extensionIdentity := common.NamespacedResource("v1", "ConfigMap", "extension-system", "extension-config")
	if _, err := client.Get(extensionIdentity); !apierrors.IsNotFound(err) {
		t.Fatalf("Get() after delete = %v, want NotFound", err)
	}
}
