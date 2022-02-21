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
	"os"
	"testing"

	mf "github.com/manifestival/manifestival"
	fake "github.com/manifestival/manifestival/fake"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"knative.dev/operator/pkg/apis/operator/base"
	"knative.dev/operator/pkg/apis/operator/v1beta1"
	util "knative.dev/operator/pkg/reconciler/common/testing"
)

func TestStagesExecute(t *testing.T) {
	koPath := "testdata/kodata"
	os.Setenv(KoEnvKey, koPath)
	defer os.Unsetenv(KoEnvKey)

	tests := []struct {
		name                  string
		component             base.KComponent
		expectedManifestsPath string
	}{{
		name: "knative-serving with additional manifests",
		component: &v1beta1.KnativeServing{
			Spec: v1beta1.KnativeServingSpec{
				CommonSpec: base.CommonSpec{
					Version: "1.0.0",
					AdditionalManifests: []base.Manifest{{
						Url: "testdata/kodata/additional-manifests/additional-resource.yaml",
					}},
				},
			},
		},
		expectedManifestsPath: os.Getenv(KoEnvKey) + "/knative-serving/1.0.0" + "," + os.Getenv(KoEnvKey) +
			"/additional-manifests/additional-resource.yaml",
	}, {
		name: "knative-serving with no additional manifests",
		component: &v1beta1.KnativeServing{
			Spec: v1beta1.KnativeServingSpec{
				CommonSpec: base.CommonSpec{
					Version: "0.26.1",
				},
			},
		},
		expectedManifestsPath: os.Getenv(KoEnvKey) + "/knative-serving/0.26.1",
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manifest, _ := mf.ManifestFrom(mf.Slice{})
			stages := Stages{AppendTarget, AppendAdditionalManifests}
			util.AssertEqual(t, len(manifest.Resources()), 0)
			err := stages.Execute(context.TODO(), &manifest, test.component)
			util.AssertEqual(t, err, nil)
			util.AssertEqual(t, util.DeepMatchWithPath(manifest, test.expectedManifestsPath), true)
		})
	}
}

func TestStagesExecuteWithRepetition(t *testing.T) {
	koPath := "testdata/kodata"
	os.Setenv(KoEnvKey, koPath)
	defer os.Unsetenv(KoEnvKey)

	testRepetition := []struct {
		name                      string
		component                 base.KComponent
		expectedContainingPath    string
		expectedManifestsPath     string
		expectedNotContainingPath string
	}{{
		name: "knative-serving with the same resource in additional manifests",
		component: &v1beta1.KnativeServing{
			Spec: v1beta1.KnativeServingSpec{
				CommonSpec: base.CommonSpec{
					Version: "1.0.0",
					AdditionalManifests: []base.Manifest{{
						Url: "testdata/kodata/additional-manifests-repetition/additional-resource.yaml",
					}},
				},
			},
		},
		expectedNotContainingPath: os.Getenv(KoEnvKey) + "/knative-serving/1.0.0/serving-core.yaml",
		expectedManifestsPath: os.Getenv(KoEnvKey) + "/knative-serving/1.0.0" + "," + os.Getenv(KoEnvKey) +
			"/additional-manifests-repetition/additional-resource.yaml",
		expectedContainingPath: os.Getenv(KoEnvKey) + "/additional-manifests-repetition/additional-resource.yaml",
	}}

	for _, test := range testRepetition {
		t.Run(test.name, func(t *testing.T) {
			manifest, _ := mf.ManifestFrom(mf.Slice{})
			stages := Stages{AppendTarget, AppendAdditionalManifests}
			util.AssertEqual(t, len(manifest.Resources()), 0)
			err := stages.Execute(context.TODO(), &manifest, test.component)
			util.AssertEqual(t, err, nil)
			// The expected manifests are not 100% identical to the actual manifests, since the additional manifests
			// have the repeated resource.
			util.AssertEqual(t, util.DeepMatchWithPath(manifest, test.expectedManifestsPath), false)
			// The actual manifests match in terms of name, namespace, group and kind.
			util.AssertEqual(t, util.ResourceMatchWithPath(manifest, test.expectedManifestsPath), true)
			// The actual manifests contain every resource available in the expectedContainingPath.
			util.AssertEqual(t, util.ResourceContainingWithPath(manifest, test.expectedContainingPath), true)
			// The actual manifests do not contain the resource available in the expectedNotContainingPath.
			util.AssertEqual(t, util.ResourceContainingWithPath(manifest, test.expectedNotContainingPath), false)
		})
	}
}

func TestStagesExecuteInstalledManifests(t *testing.T) {
	koPath := "testdata/kodata"
	os.Setenv(KoEnvKey, koPath)
	defer os.Unsetenv(KoEnvKey)

	testRepetition := []struct {
		name                  string
		component             base.KComponent
		expectedManifestsPath string
	}{{
		name: "knative-serving with no status.manifests",
		component: &v1beta1.KnativeServing{
			Spec: v1beta1.KnativeServingSpec{
				CommonSpec: base.CommonSpec{
					Version: "0.26.1",
				},
			},
		},
		expectedManifestsPath: os.Getenv(KoEnvKey) + "/knative-serving/0.26.1",
	}, {
		name: "knative-serving with status.manifests",
		component: &v1beta1.KnativeServing{
			Spec: v1beta1.KnativeServingSpec{
				CommonSpec: base.CommonSpec{
					Version: "0.26.1",
				},
			},
			Status: v1beta1.KnativeServingStatus{
				Version: "0.26.1",
				Manifests: []string{
					os.Getenv(KoEnvKey) + "/knative-serving/0.26.1",
					os.Getenv(KoEnvKey) + "/additional-manifests/additional-resource.yaml",
				},
			},
		},
		expectedManifestsPath: os.Getenv(KoEnvKey) + "/knative-serving/0.26.1" + "," + os.Getenv(KoEnvKey) +
			"/additional-manifests/additional-resource.yaml",
	}, {
		name: "knative-serving with the additional manifests in spec",
		component: &v1beta1.KnativeServing{
			Spec: v1beta1.KnativeServingSpec{
				CommonSpec: base.CommonSpec{
					Version: "0.26.1",
					AdditionalManifests: []base.Manifest{{
						Url: os.Getenv(KoEnvKey) + "/additional-manifests/additional-resource.yaml",
					}},
				},
			},
		},
		expectedManifestsPath: os.Getenv(KoEnvKey) + "/knative-serving/0.26.1",
	}, {
		name: "knative-serving with the additional manifests in spec",
		component: &v1beta1.KnativeServing{
			Spec: v1beta1.KnativeServingSpec{
				CommonSpec: base.CommonSpec{
					Version: "0.26.1",
					Manifests: []base.Manifest{{
						Url: os.Getenv(KoEnvKey) + "/knative-serving/0.26.1",
					}},
					AdditionalManifests: []base.Manifest{{
						Url: os.Getenv(KoEnvKey) + "/additional-manifests/additional-resource.yaml",
					}},
				},
			},
		},
		expectedManifestsPath: os.Getenv(KoEnvKey) + "/knative-serving/0.26.1",
	}}

	for _, test := range testRepetition {
		t.Run(test.name, func(t *testing.T) {
			manifest, _ := mf.ManifestFrom(mf.Slice{})
			stages := Stages{AppendInstalled}
			util.AssertEqual(t, len(manifest.Resources()), 0)
			err := stages.Execute(context.TODO(), &manifest, test.component)
			util.AssertEqual(t, err, nil)
			util.AssertEqual(t, util.DeepMatchWithPath(manifest, test.expectedManifestsPath), true)
		})
	}
}

func TestDeleteObsoleteResources(t *testing.T) {
	os.Setenv(KoEnvKey, "testdata/kodata")
	defer os.Unsetenv(KoEnvKey)
	client := fake.New()
	manifest, err := mf.NewManifest("testdata/manifest.yaml", mf.UseClient(client))
	if err != nil {
		t.Error(err)
	}
	// Save the manifest resources
	if err := manifest.Apply(); err != nil {
		t.Error(err)
	}
	// Grab the ConfigMaps, ensure we have at least 1
	cms := manifest.Filter(mf.ByKind("ConfigMap")).Resources()
	if len(cms) == 0 {
		t.Error("Where'd all the ConfigMaps go?!")
	}
	// Verify they exist in the "database"
	for _, cm := range cms {
		if _, err := manifest.Client.Get(&cm); err != nil {
			t.Error(err)
		}
	}
	deleteObsoleteResources := DeleteObsoleteResources(context.TODO(), &v1beta1.KnativeServing{},
		func(context.Context, base.KComponent) (*mf.Manifest, error) {
			return &manifest, nil
		})
	nocms := manifest.Filter(mf.Not(mf.ByKind("ConfigMap")))
	deleteObsoleteResources(context.TODO(), &nocms, nil)
	// Now verify all the ConfigMaps are gone
	for _, cm := range cms {
		if _, err := manifest.Client.Get(&cm); !errors.IsNotFound(err) {
			t.Errorf("ConfigMap %s should've been deleted!", cm.GetName())
		}
	}
	// And verify everything else is still there
	for _, cm := range nocms.Resources() {
		if _, err := manifest.Client.Get(&cm); err != nil {
			t.Error(err)
		}
	}
	// Now verify CRD's don't get deleted
	v1crds, _ := manifest.Transform(func(u *unstructured.Unstructured) error {
		if u.GetKind() == "CustomResourceDefinition" {
			u.SetAPIVersion("apiextensions.k8s.io/v1")
		}
		return nil
	})
	deleteObsoleteResources(context.TODO(), &v1crds, nil)
	// And verify the old ones are still there
	for _, cm := range manifest.Filter(mf.CRDs).Resources() {
		if _, err := manifest.Client.Get(&cm); err != nil {
			t.Error(err)
		}
	}
}
