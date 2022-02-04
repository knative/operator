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

package v1alpha1

import (
	"context"

	"knative.dev/operator/pkg/apis/operator/base"
	"knative.dev/operator/pkg/apis/operator/v1beta1"
	"knative.dev/pkg/apis"
)

func convertFromSourceConfigsBeta(ke *v1beta1.KnativeEventing) *SourceConfigs {
	ceph := base.CephSourceConfiguration{}
	github := base.GithubSourceConfiguration{}
	gitlab := base.GitlabSourceConfiguration{}
	kafka := base.KafkaSourceConfiguration{}
	natss := base.NatssSourceConfiguration{}
	rabbitmq := base.RabbitmqSourceConfiguration{}
	redis := base.RedisSourceConfiguration{}

	if ke.Spec.Source != nil {
		ceph = ke.Spec.Source.Ceph
		github = ke.Spec.Source.Github
		gitlab = ke.Spec.Source.Gitlab
		kafka = ke.Spec.Source.Kafka
		natss = ke.Spec.Source.Natss
		rabbitmq = ke.Spec.Source.Rabbitmq
		redis = ke.Spec.Source.Redis
	}

	return &SourceConfigs{
		Ceph:     ceph,
		Github:   github,
		Gitlab:   gitlab,
		Kafka:    kafka,
		Natss:    natss,
		Rabbitmq: rabbitmq,
		Redis:    redis,
	}
}

func convertToSourceConfigs(ke *KnativeEventing) *v1beta1.SourceConfigs {
	ceph := base.CephSourceConfiguration{}
	github := base.GithubSourceConfiguration{}
	gitlab := base.GitlabSourceConfiguration{}
	kafka := base.KafkaSourceConfiguration{}
	natss := base.NatssSourceConfiguration{}
	rabbitmq := base.RabbitmqSourceConfiguration{}
	redis := base.RedisSourceConfiguration{}

	if ke.Spec.Source != nil {
		ceph = ke.Spec.Source.Ceph
		github = ke.Spec.Source.Github
		gitlab = ke.Spec.Source.Gitlab
		kafka = ke.Spec.Source.Kafka
		natss = ke.Spec.Source.Natss
		rabbitmq = ke.Spec.Source.Rabbitmq
		redis = ke.Spec.Source.Redis
	}

	return &v1beta1.SourceConfigs{
		Ceph:     ceph,
		Github:   github,
		Gitlab:   gitlab,
		Kafka:    kafka,
		Natss:    natss,
		Rabbitmq: rabbitmq,
		Redis:    redis,
	}
}

// ConvertTo implements apis.Convertible
// Converts source from v1alpha1.KnativeEventing into a higher version.
func (ke *KnativeEventing) ConvertTo(ctx context.Context, obj apis.Convertible) error {
	switch sink := obj.(type) {
	case *v1beta1.KnativeEventing:
		mergedDeploymentOverride := ConvertToDeploymentOverride(ke)
		sourceConfigs := convertToSourceConfigs(ke)
		sink.ObjectMeta = ke.ObjectMeta
		sink.Status = v1beta1.KnativeEventingStatus{
			Status:    ke.Status.Status,
			Version:   ke.Status.Version,
			Manifests: ke.Status.Manifests,
		}
		sink.Spec = v1beta1.KnativeEventingSpec{
			DefaultBrokerClass:       ke.Spec.DefaultBrokerClass,
			SinkBindingSelectionMode: ke.Spec.SinkBindingSelectionMode,
			Source:                   sourceConfigs,
			CommonSpec: base.CommonSpec{
				Config:              ke.Spec.CommonSpec.Config,
				Registry:            ke.Spec.CommonSpec.Registry,
				DeploymentOverride:  mergedDeploymentOverride,
				Version:             ke.Spec.CommonSpec.Version,
				Manifests:           ke.Spec.CommonSpec.Manifests,
				AdditionalManifests: ke.Spec.CommonSpec.AdditionalManifests,
				HighAvailability:    ke.Spec.CommonSpec.HighAvailability,
			},
		}

		return nil
	default:
		return apis.ConvertToViaProxy(ctx, ke, &v1beta1.KnativeEventing{}, sink)
	}
}

// ConvertFrom implements apis.Convertible
// Converts source from a higher version into v1alpha1.KnativeEventing
func (ke *KnativeEventing) ConvertFrom(ctx context.Context, obj apis.Convertible) error {
	switch source := obj.(type) {
	case *v1beta1.KnativeEventing:
		sourceConfigs := convertFromSourceConfigsBeta(source)
		ke.ObjectMeta = source.ObjectMeta
		ke.Status = KnativeEventingStatus{
			Status:    source.Status.Status,
			Version:   source.Status.Version,
			Manifests: source.Status.Manifests,
		}

		ke.Spec = KnativeEventingSpec{
			DefaultBrokerClass:       source.Spec.DefaultBrokerClass,
			SinkBindingSelectionMode: source.Spec.SinkBindingSelectionMode,
			Source:                   sourceConfigs,
			CommonSpec: base.CommonSpec{
				Config:              source.Spec.CommonSpec.Config,
				Registry:            source.Spec.CommonSpec.Registry,
				DeploymentOverride:  source.Spec.CommonSpec.DeploymentOverride,
				Version:             source.Spec.CommonSpec.Version,
				Manifests:           source.Spec.CommonSpec.Manifests,
				AdditionalManifests: source.Spec.CommonSpec.AdditionalManifests,
				HighAvailability:    source.Spec.CommonSpec.HighAvailability,
			},
		}

		return nil
	default:
		return apis.ConvertFromViaProxy(ctx, source, &v1beta1.KnativeEventing{}, ke)
	}
}
