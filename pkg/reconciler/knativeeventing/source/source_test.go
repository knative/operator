/*
Copyright 2021 The Knative Authors

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

package source

import (
	"context"
	"fmt"
	"os"
	"testing"

	mf "github.com/manifestival/manifestival"
	"knative.dev/operator/pkg/apis/operator/base"
	eventingv1beta1 "knative.dev/operator/pkg/apis/operator/v1beta1"
	"knative.dev/operator/pkg/reconciler/common"
	util "knative.dev/operator/pkg/reconciler/common/testing"
)

func TestAppendAllSources(t *testing.T) {
	os.Setenv(common.KoEnvKey, "testdata/kodata")
	defer os.Unsetenv(common.KoEnvKey)

	tests := []struct {
		name                string
		instance            eventingv1beta1.KnativeEventing
		expectedIngressPath string
		expectedErr         error
	}{{
		name: "Available Amazon SQS, Redis, Ceph, Couchdb and GitHub as the target sources",
		instance: eventingv1beta1.KnativeEventing{
			Spec: eventingv1beta1.KnativeEventingSpec{
				Source: &eventingv1beta1.SourceConfigs{
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
			Status: eventingv1beta1.KnativeEventingStatus{
				Version: "0.22",
			},
		},
		expectedIngressPath: os.Getenv(common.KoEnvKey) + "/eventing-source/0.22/ceph" + common.COMMA +
			os.Getenv(common.KoEnvKey) + "/eventing-source/0.22/github" + common.COMMA +
			os.Getenv(common.KoEnvKey) + "/eventing-source/0.22/gitlab" + common.COMMA +
			os.Getenv(common.KoEnvKey) + "/eventing-source/0.22/kafka" + common.COMMA +
			os.Getenv(common.KoEnvKey) + "/eventing-source/0.22/redis" + common.COMMA +
			os.Getenv(common.KoEnvKey) + "/eventing-source/0.22/rabbitmq",
		expectedErr: nil,
	}, {
		name: "Available GitLab, Kafka, Rabbitmq and Prometheus as the target sources",
		instance: eventingv1beta1.KnativeEventing{
			Spec: eventingv1beta1.KnativeEventingSpec{
				Source: &eventingv1beta1.SourceConfigs{
					Kafka: base.KafkaSourceConfiguration{
						Enabled: true,
					},
					Gitlab: base.GitlabSourceConfiguration{
						Enabled: true,
					},
					Rabbitmq: base.RabbitmqSourceConfiguration{
						Enabled: true,
					},
				},
			},
			Status: eventingv1beta1.KnativeEventingStatus{
				Version: "0.22",
			},
		},
		expectedIngressPath: os.Getenv(common.KoEnvKey) + "/eventing-source/0.22/ceph" + common.COMMA +
			os.Getenv(common.KoEnvKey) + "/eventing-source/0.22/github" + common.COMMA +
			os.Getenv(common.KoEnvKey) + "/eventing-source/0.22/gitlab" + common.COMMA +
			os.Getenv(common.KoEnvKey) + "/eventing-source/0.22/kafka" + common.COMMA +
			os.Getenv(common.KoEnvKey) + "/eventing-source/0.22/redis" + common.COMMA +
			os.Getenv(common.KoEnvKey) + "/eventing-source/0.22/rabbitmq",
		expectedErr: nil,
	}, {
		name: "No source is enabled",
		instance: eventingv1beta1.KnativeEventing{
			Spec: eventingv1beta1.KnativeEventingSpec{},
			Status: eventingv1beta1.KnativeEventingStatus{
				Version: "0.23",
			},
		},
		expectedIngressPath: os.Getenv(common.KoEnvKey) + "/eventing-source/0.23/ceph" + common.COMMA +
			os.Getenv(common.KoEnvKey) + "/eventing-source/0.23/github" + common.COMMA +
			os.Getenv(common.KoEnvKey) + "/eventing-source/0.23/gitlab" + common.COMMA +
			os.Getenv(common.KoEnvKey) + "/eventing-source/0.23/kafka" + common.COMMA +
			os.Getenv(common.KoEnvKey) + "/eventing-source/0.23/redis" + common.COMMA +
			os.Getenv(common.KoEnvKey) + "/eventing-source/0.23/rabbitmq",
		expectedErr: nil,
	}, {
		name: "Unavailable eventing source",
		instance: eventingv1beta1.KnativeEventing{
			Spec: eventingv1beta1.KnativeEventingSpec{},
			Status: eventingv1beta1.KnativeEventingStatus{
				Version: "0.21",
			},
		},
		expectedIngressPath: os.Getenv(common.KoEnvKey) + "/eventing-source/empty.yaml",
		expectedErr:         nil,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifest, _ := mf.ManifestFrom(mf.Slice{})
			err := AppendAllSources(context.TODO(), &manifest, &tt.instance)
			if err != nil {
				util.AssertEqual(t, err.Error(), tt.expectedErr.Error())
				util.AssertEqual(t, len(manifest.Resources()), 0)
			} else {
				util.AssertEqual(t, err, tt.expectedErr)
				util.AssertEqual(t, util.DeepMatchWithPath(manifest, tt.expectedIngressPath), true)
			}
		})
	}
}

func TestAppendTargetSources(t *testing.T) {
	os.Setenv(common.KoEnvKey, "testdata/kodata")
	defer os.Unsetenv(common.KoEnvKey)

	tests := []struct {
		name                string
		instance            eventingv1beta1.KnativeEventing
		expectedIngressPath string
		expectedErr         error
	}{{
		name: "Available Amazon SQS, Redis, Ceph, Couchdb and GitHub as the target sources",
		instance: eventingv1beta1.KnativeEventing{
			Spec: eventingv1beta1.KnativeEventingSpec{
				CommonSpec: base.CommonSpec{
					Version: "0.22",
				},
				Source: &eventingv1beta1.SourceConfigs{
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
		},
		expectedIngressPath: os.Getenv(common.KoEnvKey) + "/eventing-source/0.22/ceph" + common.COMMA +
			os.Getenv(common.KoEnvKey) + "/eventing-source/0.22/github" + common.COMMA +
			os.Getenv(common.KoEnvKey) + "/eventing-source/0.22/redis",
		expectedErr: nil,
	}, {
		name: "Available GitLab, Kafka, Rabbitmq and Prometheus as the target sources",
		instance: eventingv1beta1.KnativeEventing{
			Spec: eventingv1beta1.KnativeEventingSpec{
				CommonSpec: base.CommonSpec{
					Version: "0.22",
				},
				Source: &eventingv1beta1.SourceConfigs{
					Kafka: base.KafkaSourceConfiguration{
						Enabled: true,
					},
					Gitlab: base.GitlabSourceConfiguration{
						Enabled: true,
					},
					Rabbitmq: base.RabbitmqSourceConfiguration{
						Enabled: true,
					},
				},
			},
		},
		expectedIngressPath: os.Getenv(common.KoEnvKey) + "/eventing-source/0.22/kafka" + common.COMMA +
			os.Getenv(common.KoEnvKey) + "/eventing-source/0.22/gitlab" + common.COMMA +
			os.Getenv(common.KoEnvKey) + "/eventing-source/0.22/rabbitmq",
		expectedErr: nil,
	}, {
		name: "Unavailable target source",
		instance: eventingv1beta1.KnativeEventing{
			Spec: eventingv1beta1.KnativeEventingSpec{
				CommonSpec: base.CommonSpec{
					Version: "0.12.1",
				},
				Source: &eventingv1beta1.SourceConfigs{
					Ceph: base.CephSourceConfiguration{
						Enabled: true,
					},
				},
			},
		},
		expectedErr: fmt.Errorf("stat testdata/kodata/eventing-source/0.12/ceph: no such file or directory"),
	}, {
		name: "Unavailable target source with spec.manifests",
		instance: eventingv1beta1.KnativeEventing{
			Spec: eventingv1beta1.KnativeEventingSpec{
				CommonSpec: base.CommonSpec{
					Version: "0.12.1",
					Manifests: []base.Manifest{{
						Url: "testdata/kodata/eventing-source/empty.yaml",
					}},
				},
				Source: &eventingv1beta1.SourceConfigs{
					Ceph: base.CephSourceConfiguration{
						Enabled: true,
					},
				},
			},
		},
		expectedErr: nil,
	}, {
		name: "Get the latest target source when the directory latest is unavailable",
		instance: eventingv1beta1.KnativeEventing{
			Spec: eventingv1beta1.KnativeEventingSpec{
				CommonSpec: base.CommonSpec{
					Version: "latest",
				},
				Source: &eventingv1beta1.SourceConfigs{
					Ceph: base.CephSourceConfiguration{
						Enabled: true,
					},
				},
			},
		},
		expectedIngressPath: os.Getenv(common.KoEnvKey) + "/eventing-source/0.23/ceph",
		expectedErr:         nil,
	}, {
		name: "No source is enabled",
		instance: eventingv1beta1.KnativeEventing{
			Spec: eventingv1beta1.KnativeEventingSpec{
				CommonSpec: base.CommonSpec{
					Version: "0.23",
				},
			},
		},
		expectedIngressPath: os.Getenv(common.KoEnvKey) + "/eventing-source/empty.yaml",
		expectedErr:         nil,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifest, _ := mf.ManifestFrom(mf.Slice{})
			err := AppendTargetSources(context.TODO(), &manifest, &tt.instance)
			if err != nil {
				util.AssertEqual(t, err.Error(), tt.expectedErr.Error())
				util.AssertEqual(t, len(manifest.Resources()), 0)
			} else {
				util.AssertEqual(t, err, tt.expectedErr)
				util.AssertEqual(t, util.DeepMatchWithPath(manifest, tt.expectedIngressPath), true)
			}
		})
	}
}

func TestGetSourcePath(t *testing.T) {
	os.Setenv(common.KoEnvKey, "testdata/kodata")
	defer os.Unsetenv(common.KoEnvKey)

	tests := []struct {
		name               string
		version            string
		instance           eventingv1beta1.KnativeEventing
		expectedSourcePath string
	}{{
		name:    "Available Amazon SQS, Redis, Ceph, Couchdb and GitHub as the target sources",
		version: "0.22.1",
		instance: eventingv1beta1.KnativeEventing{
			Spec: eventingv1beta1.KnativeEventingSpec{
				CommonSpec: base.CommonSpec{
					Version: "0.22",
				},
				Source: &eventingv1beta1.SourceConfigs{
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
		},
		expectedSourcePath: os.Getenv(common.KoEnvKey) + "/eventing-source/0.22/ceph" + common.COMMA +
			os.Getenv(common.KoEnvKey) + "/eventing-source/0.22/github" + common.COMMA +
			os.Getenv(common.KoEnvKey) + "/eventing-source/0.22/redis",
	}, {
		name:    "Available GitLab, Kafka, Rabbitmq and Prometheus as the target sources",
		version: "0.22.1",
		instance: eventingv1beta1.KnativeEventing{
			Spec: eventingv1beta1.KnativeEventingSpec{
				CommonSpec: base.CommonSpec{
					Version: "0.22",
				},
				Source: &eventingv1beta1.SourceConfigs{
					Kafka: base.KafkaSourceConfiguration{
						Enabled: true,
					},
					Gitlab: base.GitlabSourceConfiguration{
						Enabled: true,
					},
					Rabbitmq: base.RabbitmqSourceConfiguration{
						Enabled: true,
					},
				},
			},
		},
		expectedSourcePath: os.Getenv(common.KoEnvKey) + "/eventing-source/0.22/gitlab" + common.COMMA +
			os.Getenv(common.KoEnvKey) + "/eventing-source/0.22/kafka" + common.COMMA +
			os.Getenv(common.KoEnvKey) + "/eventing-source/0.22/rabbitmq",
	}, {
		name:    "No source is enabled",
		version: "0.23.0",
		instance: eventingv1beta1.KnativeEventing{
			Spec: eventingv1beta1.KnativeEventingSpec{
				CommonSpec: base.CommonSpec{
					Version: "0.23",
				},
			},
		},
		expectedSourcePath: "",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := GetSourcePath(tt.version, &tt.instance)
			util.AssertEqual(t, path, tt.expectedSourcePath)
		})
	}
}
