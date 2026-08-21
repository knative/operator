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

package knativeeventing

import (
	"context"
	"strings"
	"testing"

	mf "github.com/manifestival/manifestival"
	mffake "github.com/manifestival/manifestival/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"knative.dev/operator/pkg/apis/operator/base"
	"knative.dev/operator/pkg/apis/operator/v1beta1"
	"knative.dev/operator/pkg/reconciler/common"
)

func TestPatchesRunAfterTLSResourceFiltering(t *testing.T) {
	certificate := unstructured.Unstructured{}
	certificate.SetAPIVersion("cert-manager.io/v1")
	certificate.SetKind("Certificate")
	certificate.SetNamespace("knative-eventing")
	certificate.SetName("eventing-webhook")
	manifest, err := mf.ManifestFrom(
		mf.Slice([]unstructured.Unstructured{certificate}),
		mf.UseClient(mffake.New()),
	)
	if err != nil {
		t.Fatalf("ManifestFrom() = %v", err)
	}
	instance := &v1beta1.KnativeEventing{
		ObjectMeta: metav1.ObjectMeta{Namespace: "knative-eventing"},
		Spec: v1beta1.KnativeEventingSpec{CommonSpec: base.CommonSpec{
			Patches: []base.ResourcePatch{{
				Target: base.PatchTarget{
					APIVersion: "cert-manager.io/v1",
					Kind:       "Certificate",
					Name:       "eventing-webhook",
				},
				Patch: base.PatchSpec{Type: base.MergePatchType, Content: `{}`},
			}},
		}},
	}

	patchStages := common.NewResourcePatchStages()
	stages := common.Stages{(&Reconciler{}).handleTLSResources, patchStages.Apply}
	if _, err := stages.Execute(context.Background(), &manifest, instance); err == nil || !strings.Contains(err.Error(), "did not match any generated resource") {
		t.Fatalf("Stages.Execute() = %v, want missing patch target error", err)
	}
}
