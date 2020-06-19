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
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
	util "knative.dev/operator/pkg/reconciler/common/testing"
)

func TestStagesExecute(t *testing.T) {
	os.Setenv(KoEnvKey, "testdata/kodata")
	defer os.Unsetenv(KoEnvKey)
	manifest, _ := mf.ManifestFrom(mf.Slice{})
	stages := Stages{AppendTarget, AppendInstalled}
	util.AssertEqual(t, len(manifest.Resources()), 0)
	stages.Execute(context.TODO(), &manifest, &v1alpha1.KnativeServing{})
	util.AssertEqual(t, len(manifest.Resources()), 2)
}
