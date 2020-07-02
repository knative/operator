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

	. "github.com/manifestival/manifestival"
	"github.com/manifestival/manifestival/pkg/fake"
	. "github.com/manifestival/manifestival/pkg/filter"
	. "github.com/manifestival/manifestival/pkg/sources"
	"k8s.io/apimachinery/pkg/api/errors"
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
	util "knative.dev/operator/pkg/reconciler/common/testing"
)

func TestStagesExecute(t *testing.T) {
	os.Setenv(KoEnvKey, "testdata/kodata")
	defer os.Unsetenv(KoEnvKey)
	manifest, _ := ManifestFrom(Slice{})
	stages := Stages{AppendTarget, AppendInstalled}
	util.AssertEqual(t, len(manifest.Resources()), 0)
	stages.Execute(context.TODO(), &manifest, &v1alpha1.KnativeServing{})
	util.AssertEqual(t, len(manifest.Resources()), 2)
}

func TestDeleteObsoleteResources(t *testing.T) {
	os.Setenv(KoEnvKey, "testdata/kodata")
	defer os.Unsetenv(KoEnvKey)
	client := fake.New()
	manifest, err := NewManifest("testdata/manifest.yaml", UseClient(client))
	if err != nil {
		t.Error(err)
	}
	// Save the manifest resources
	if err := manifest.Apply(); err != nil {
		t.Error(err)
	}
	// Grab the ConfigMaps, ensure we have at least 1
	cms := manifest.Filter(ByKind("ConfigMap")).Resources()
	if len(cms) == 0 {
		t.Error("Where'd all the ConfigMaps go?!")
	}
	// Verify they exist in the "database"
	for _, cm := range cms {
		if _, err := manifest.Client.Get(&cm); err != nil {
			t.Error(err)
		}
	}
	stage := DeleteObsoleteResources(context.TODO(), &v1alpha1.KnativeServing{},
		func(context.Context, v1alpha1.KComponent) (*Manifest, error) {
			return &manifest, nil
		})
	nocms := manifest.Filter(Not(ByKind("ConfigMap")))
	stage(context.TODO(), &nocms, nil)
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

}
