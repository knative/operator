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
	eventingv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
	"knative.dev/operator/pkg/reconciler/common"
	util "knative.dev/operator/pkg/reconciler/common/testing"
)

func TestAppendInstalledSources(t *testing.T) {
	os.Setenv(common.KoEnvKey, "testdata/kodata")
	defer os.Unsetenv(common.KoEnvKey)

	tests := []struct {
		name                string
		instance            eventingv1alpha1.KnativeEventing
		expectedIngressPath string
		expectedErr         error
	}{{
		name: "Available Amazon SQS, Redis, Ceph, Couchdb and GitHub as the target sources",
		instance: eventingv1alpha1.KnativeEventing{
			Spec: eventingv1alpha1.KnativeEventingSpec{
				Source: &eventingv1alpha1.SourceConfigs{
					Awssqs: eventingv1alpha1.AwssqsSourceConfiguration{
						Enabled: true,
					},
					Ceph: eventingv1alpha1.CephSourceConfiguration{
						Enabled: true,
					},
					Github: eventingv1alpha1.GithubSourceConfiguration{
						Enabled: true,
					},
					Redis: eventingv1alpha1.RedisSourceConfiguration{
						Enabled: true,
					},
					Couchdb: eventingv1alpha1.CouchdbSourceConfiguration{
						Enabled: true,
					},
				},
			},
			Status: eventingv1alpha1.KnativeEventingStatus{
				Version: "0.22",
			},
		},
		expectedIngressPath: os.Getenv(common.KoEnvKey) + "/eventing-source/0.22/awssqs" + common.COMMA +
			os.Getenv(common.KoEnvKey) + "/eventing-source/0.22/ceph" + common.COMMA +
			os.Getenv(common.KoEnvKey) + "/eventing-source/0.22/github" + common.COMMA +
			os.Getenv(common.KoEnvKey) + "/eventing-source/0.22/couchdb" + common.COMMA +
			os.Getenv(common.KoEnvKey) + "/eventing-source/0.22/redis",
		expectedErr: nil,
	}, {
		name: "Available GitLab, Kafka, NATSS, Rabbitmq and Prometheus as the target sources",
		instance: eventingv1alpha1.KnativeEventing{
			Spec: eventingv1alpha1.KnativeEventingSpec{
				Source: &eventingv1alpha1.SourceConfigs{
					Natss: eventingv1alpha1.NatssSourceConfiguration{
						Enabled: true,
					},
					Kafka: eventingv1alpha1.KafkaSourceConfiguration{
						Enabled: true,
					},
					Gitlab: eventingv1alpha1.GitlabSourceConfiguration{
						Enabled: true,
					},
					Prometheus: eventingv1alpha1.PrometheusSourceConfiguration{
						Enabled: true,
					},
					Rabbitmq: eventingv1alpha1.RabbitmqSourceConfiguration{
						Enabled: true,
					},
				},
			},
			Status: eventingv1alpha1.KnativeEventingStatus{
				Version: "0.22",
			},
		},
		expectedIngressPath: os.Getenv(common.KoEnvKey) + "/eventing-source/0.22/natss" + common.COMMA +
			os.Getenv(common.KoEnvKey) + "/eventing-source/0.22/kafka" + common.COMMA +
			os.Getenv(common.KoEnvKey) + "/eventing-source/0.22/gitlab" + common.COMMA +
			os.Getenv(common.KoEnvKey) + "/eventing-source/0.22/prometheus" + common.COMMA +
			os.Getenv(common.KoEnvKey) + "/eventing-source/0.22/rabbitmq",
		expectedErr: nil,
	}, {
		name: "No source is enabled",
		instance: eventingv1alpha1.KnativeEventing{
			Spec: eventingv1alpha1.KnativeEventingSpec{},
			Status: eventingv1alpha1.KnativeEventingStatus{
				Version: "0.23",
			},
		},
		expectedIngressPath: os.Getenv(common.KoEnvKey) + "/eventing-source/empty.yaml",
		expectedErr:         nil,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifest, _ := mf.ManifestFrom(mf.Slice{})
			err := AppendInstalledSources(context.TODO(), &manifest, &tt.instance)
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
		instance            eventingv1alpha1.KnativeEventing
		expectedIngressPath string
		expectedErr         error
	}{{
		name: "Available Amazon SQS, Redis, Ceph, Couchdb and GitHub as the target sources",
		instance: eventingv1alpha1.KnativeEventing{
			Spec: eventingv1alpha1.KnativeEventingSpec{
				CommonSpec: eventingv1alpha1.CommonSpec{
					Version: "0.22",
				},
				Source: &eventingv1alpha1.SourceConfigs{
					Awssqs: eventingv1alpha1.AwssqsSourceConfiguration{
						Enabled: true,
					},
					Ceph: eventingv1alpha1.CephSourceConfiguration{
						Enabled: true,
					},
					Github: eventingv1alpha1.GithubSourceConfiguration{
						Enabled: true,
					},
					Redis: eventingv1alpha1.RedisSourceConfiguration{
						Enabled: true,
					},
					Couchdb: eventingv1alpha1.CouchdbSourceConfiguration{
						Enabled: true,
					},
				},
			},
		},
		expectedIngressPath: os.Getenv(common.KoEnvKey) + "/eventing-source/0.22/awssqs" + common.COMMA +
			os.Getenv(common.KoEnvKey) + "/eventing-source/0.22/ceph" + common.COMMA +
			os.Getenv(common.KoEnvKey) + "/eventing-source/0.22/github" + common.COMMA +
			os.Getenv(common.KoEnvKey) + "/eventing-source/0.22/couchdb" + common.COMMA +
			os.Getenv(common.KoEnvKey) + "/eventing-source/0.22/redis",
		expectedErr: nil,
	}, {
		name: "Available GitLab, Kafka, NATSS, Rabbitmq and Prometheus as the target sources",
		instance: eventingv1alpha1.KnativeEventing{
			Spec: eventingv1alpha1.KnativeEventingSpec{
				CommonSpec: eventingv1alpha1.CommonSpec{
					Version: "0.22",
				},
				Source: &eventingv1alpha1.SourceConfigs{
					Natss: eventingv1alpha1.NatssSourceConfiguration{
						Enabled: true,
					},
					Kafka: eventingv1alpha1.KafkaSourceConfiguration{
						Enabled: true,
					},
					Gitlab: eventingv1alpha1.GitlabSourceConfiguration{
						Enabled: true,
					},
					Prometheus: eventingv1alpha1.PrometheusSourceConfiguration{
						Enabled: true,
					},
					Rabbitmq: eventingv1alpha1.RabbitmqSourceConfiguration{
						Enabled: true,
					},
				},
			},
		},
		expectedIngressPath: os.Getenv(common.KoEnvKey) + "/eventing-source/0.22/natss" + common.COMMA +
			os.Getenv(common.KoEnvKey) + "/eventing-source/0.22/kafka" + common.COMMA +
			os.Getenv(common.KoEnvKey) + "/eventing-source/0.22/gitlab" + common.COMMA +
			os.Getenv(common.KoEnvKey) + "/eventing-source/0.22/prometheus" + common.COMMA +
			os.Getenv(common.KoEnvKey) + "/eventing-source/0.22/rabbitmq",
		expectedErr: nil,
	}, {
		name: "Unavailable target source",
		instance: eventingv1alpha1.KnativeEventing{
			Spec: eventingv1alpha1.KnativeEventingSpec{
				CommonSpec: eventingv1alpha1.CommonSpec{
					Version: "0.12.1",
				},
				Source: &eventingv1alpha1.SourceConfigs{
					Awssqs: eventingv1alpha1.AwssqsSourceConfiguration{
						Enabled: true,
					},
				},
			},
		},
		expectedErr: fmt.Errorf("stat testdata/kodata/eventing-source/0.12/awssqs: no such file or directory"),
	}, {
		name: "Get the latest target source when the directory latest is unavailable",
		instance: eventingv1alpha1.KnativeEventing{
			Spec: eventingv1alpha1.KnativeEventingSpec{
				CommonSpec: eventingv1alpha1.CommonSpec{
					Version: "latest",
				},
				Source: &eventingv1alpha1.SourceConfigs{
					Awssqs: eventingv1alpha1.AwssqsSourceConfiguration{
						Enabled: true,
					},
				},
			},
		},
		expectedIngressPath: os.Getenv(common.KoEnvKey) + "/eventing-source/0.23/awssqs",
		expectedErr:         nil,
	}, {
		name: "No source is enabled",
		instance: eventingv1alpha1.KnativeEventing{
			Spec: eventingv1alpha1.KnativeEventingSpec{
				CommonSpec: eventingv1alpha1.CommonSpec{
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
