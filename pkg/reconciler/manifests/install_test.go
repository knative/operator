/*
Copyright 2022 The Knative Authors

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

package manifests

import (
	"context"
	"os"
	"testing"

	mf "github.com/manifestival/manifestival"
	"knative.dev/operator/pkg/apis/operator/base"
	"knative.dev/operator/pkg/apis/operator/v1beta1"
	"knative.dev/operator/pkg/reconciler/common"
	util "knative.dev/operator/pkg/reconciler/common/testing"
)

func TestInstall(t *testing.T) {
	os.Setenv(common.KoEnvKey, "testdata/kodata")
	defer os.Unsetenv(common.KoEnvKey)

	tests := []struct {
		name         string
		version      string
		instance     base.KComponent
		expectedPath []string
	}{{
		name:    "Knative Eventing with sources",
		version: "0.23.0",
		instance: &v1beta1.KnativeEventing{
			Spec: v1beta1.KnativeEventingSpec{
				Source: &v1beta1.SourceConfigs{
					Ceph: base.CephSourceConfiguration{
						Enabled: true,
					},
					Github: base.GithubSourceConfiguration{
						Enabled: true,
					},
					Redis: base.RedisSourceConfiguration{
						Enabled: true,
					},
				},
			},
			Status: v1beta1.KnativeEventingStatus{
				Version: "0.23",
			},
		},
		expectedPath: []string{os.Getenv(common.KoEnvKey) + "/knative-eventing/0.23.0",
			os.Getenv(common.KoEnvKey) + "/eventing-source/0.23/ceph" + common.COMMA +
				os.Getenv(common.KoEnvKey) + "/eventing-source/0.23/github" + common.COMMA +
				os.Getenv(common.KoEnvKey) + "/eventing-source/0.23/redis"},
	}, {
		name:    "Knative Eventing with no source selected",
		version: "0.23.0",
		instance: &v1beta1.KnativeEventing{
			Spec: v1beta1.KnativeEventingSpec{},
			Status: v1beta1.KnativeEventingStatus{
				Version: "0.23",
			},
		},
		expectedPath: []string{os.Getenv(common.KoEnvKey) + "/knative-eventing/0.23.0"},
	}, {
		name:    "Knative Serving with ingress",
		version: "1.9.0",
		instance: &v1beta1.KnativeServing{
			Spec: v1beta1.KnativeServingSpec{
				Ingress: &v1beta1.IngressConfigs{
					Kourier: base.KourierIngressConfiguration{
						Enabled: true,
					},
				},
			},
			Status: v1beta1.KnativeServingStatus{
				Version: "1.9.0",
			},
		},
		expectedPath: []string{os.Getenv(common.KoEnvKey) + "/knative-serving/1.9.0",
			os.Getenv(common.KoEnvKey) + "/ingress/1.9/kourier"},
	}, {
		name:    "Knative Serving with no ingress selected",
		version: "1.9.0",
		instance: &v1beta1.KnativeServing{
			Spec: v1beta1.KnativeServingSpec{},
			Status: v1beta1.KnativeServingStatus{
				Version: "1.9.0",
			},
		},
		expectedPath: []string{os.Getenv(common.KoEnvKey) + "/knative-serving/1.9.0",
			os.Getenv(common.KoEnvKey) + "/ingress/1.9/istio"},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifest, _ := mf.ManifestFrom(mf.Slice{})
			err := Install(context.TODO(), &manifest, tt.instance)
			util.AssertEqual(t, err, nil)
			err = common.InstallWebhookConfigs(context.TODO(), &manifest, tt.instance)
			util.AssertEqual(t, err, nil)
			err = SetManifestPaths(context.TODO(), &manifest, tt.instance)
			util.AssertEqual(t, err, nil)
			err = common.MarkStatusSuccess(context.TODO(), &manifest, tt.instance)
			util.AssertEqual(t, err, nil)
			util.AssertEqual(t, tt.instance.GetStatus().GetVersion(), tt.version)
			util.AssertDeepEqual(t, tt.instance.GetStatus().GetManifests(), tt.expectedPath)
		})
	}
}
