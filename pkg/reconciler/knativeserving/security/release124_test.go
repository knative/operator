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

package security

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"

	mf "github.com/manifestival/manifestival"
	"knative.dev/operator/pkg/apis/operator/base"
	"knative.dev/operator/pkg/apis/operator/v1beta1"
	"knative.dev/operator/pkg/reconciler/common"
)

func TestSecurityGuardForServing124(t *testing.T) {
	data := filepath.Join("..", "..", "..", "..", "cmd", "operator", "kodata")
	t.Setenv(common.KoEnvKey, data)
	common.ClearCache()
	t.Cleanup(common.ClearCache)
	expected, err := common.FetchManifest(filepath.Join(data, "security-guard", "0.6"))
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range []string{"1.24", "1.24.0", "v1.24.1", "latest", ""} {
		t.Run(version, func(t *testing.T) {
			instance := &v1beta1.KnativeServing{Spec: v1beta1.KnativeServingSpec{
				CommonSpec: base.CommonSpec{Version: version},
				Security:   &v1beta1.SecurityConfigs{SecurityGuard: base.SecurityGuardConfiguration{Enabled: true}},
			}}
			manifest, err := mf.ManifestFrom(mf.Slice{})
			if err != nil {
				t.Fatal(err)
			}
			if err := AppendTargetSecurity(context.Background(), &manifest, instance); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(manifest.Resources(), expected.Resources()) {
				t.Fatal("did not install the bundled Security Guard 0.6 manifests")
			}
			if len(Transformers(context.Background(), instance)) != 1 {
				t.Fatal("Security Guard transformations must remain enabled")
			}
		})
	}
}
