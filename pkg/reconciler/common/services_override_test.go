package common

import (
	"testing"

	corev1 "k8s.io/api/core/v1"

	"github.com/google/go-cmp/cmp"
	mf "github.com/manifestival/manifestival"
	"k8s.io/client-go/kubernetes/scheme"
	"knative.dev/operator/pkg/apis/operator/base"
	servingv1beta1 "knative.dev/operator/pkg/apis/operator/v1beta1"
)

type expServices struct {
	expLabels      map[string]string
	expAnnotations map[string]string
}

func TestServicesTransform(t *testing.T) {
	tests := []struct {
		name        string
		override    []base.ServiceOverride
		expServices map[string]expServices
	}{{
		name: "simple override",
		override: []base.ServiceOverride{
			{
				Name:        "controller",
				Labels:      map[string]string{"a": "b"},
				Annotations: map[string]string{"c": "d"},
			},
		},
		expServices: map[string]expServices{"controller": {
			expLabels:      map[string]string{"serving.knative.dev/release": "v0.13.0", "a": "b", "app": "controller"},
			expAnnotations: map[string]string{"c": "d"},
		}},
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manifest, err := mf.NewManifest("testdata/manifest.yaml")
			if err != nil {
				t.Fatalf("Failed to create manifest: %v", err)
			}

			ks := &servingv1beta1.KnativeServing{
				Spec: servingv1beta1.KnativeServingSpec{
					CommonSpec: base.CommonSpec{
						ServiceOverride: test.override,
					},
				},
			}

			manifest, err = manifest.Transform(ServicesTransform(ks, log))
			if err != nil {
				t.Fatalf("Failed to transform manifest: %v", err)
			}

			for expName, d := range test.expServices {
				for _, u := range manifest.Resources() {
					if u.GetKind() == "Service" && u.GetName() == expName {
						got := &corev1.Service{}
						if err := scheme.Scheme.Convert(&u, got, nil); err != nil {
							t.Fatalf("Failed to convert unstructured to deployment: %v", err)
						}

						if diff := cmp.Diff(got.GetLabels(), d.expLabels); diff != "" {
							t.Fatalf("Unexpected labels: %v", diff)
						}

						if diff := cmp.Diff(got.GetAnnotations(), d.expAnnotations); diff != "" {
							t.Fatalf("Unexpected annotations: %v", diff)
						}
					}
				}
			}
		})
	}
}
