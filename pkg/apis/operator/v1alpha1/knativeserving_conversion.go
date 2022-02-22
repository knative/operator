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

func convertFromIngressConfigsBeta(ks *v1beta1.KnativeServing) *IngressConfigs {
	if ks.Spec.Ingress != nil {
		contour := ks.Spec.Ingress.Contour
		istio := ks.Spec.Ingress.Istio
		kourier := ks.Spec.Ingress.Kourier
		return &IngressConfigs{
			Contour: contour,
			Istio:   istio,
			Kourier: kourier,
		}
	}
	return nil
}

func convertToIngressConfigs(ks *KnativeServing) *v1beta1.IngressConfigs {
	if ks.Spec.Ingress != nil {
		contour := ks.Spec.Ingress.Contour
		istio := ConvertToIstioConfig(ks)
		kourier := ks.Spec.Ingress.Kourier
		return &v1beta1.IngressConfigs{
			Contour: contour,
			Istio:   istio,
			Kourier: kourier,
		}
	}
	return nil
}

// ConvertTo implements apis.Convertible
// Converts source from v1beta1.KnativeServing into a higher version.
func (ks *KnativeServing) ConvertTo(ctx context.Context, obj apis.Convertible) error {
	switch sink := obj.(type) {
	case *v1beta1.KnativeServing:
		mergedDeploymentOverride := ConvertToDeploymentOverride(ks)
		ingressConfigs := convertToIngressConfigs(ks)
		sink.ObjectMeta = ks.ObjectMeta
		sink.Status = v1beta1.KnativeServingStatus{
			Status:    ks.Status.Status,
			Version:   ks.Status.Version,
			Manifests: ks.Status.Manifests,
		}
		sink.Spec = v1beta1.KnativeServingSpec{
			ControllerCustomCerts: ks.Spec.ControllerCustomCerts,
			Ingress:               ingressConfigs,
			CommonSpec: base.CommonSpec{
				Config:              ks.Spec.CommonSpec.Config,
				Registry:            ks.Spec.CommonSpec.Registry,
				DeploymentOverride:  mergedDeploymentOverride,
				Version:             ks.Spec.CommonSpec.Version,
				Manifests:           ks.Spec.CommonSpec.Manifests,
				AdditionalManifests: ks.Spec.CommonSpec.AdditionalManifests,
				HighAvailability:    ks.Spec.CommonSpec.HighAvailability,
			},
		}

		return nil
	default:
		return apis.ConvertToViaProxy(ctx, ks, &v1beta1.KnativeServing{}, sink)
	}
}

// ConvertFrom implements apis.Convertible
// Converts source from a higher version into v1beta1.KnativeServing
func (ks *KnativeServing) ConvertFrom(ctx context.Context, obj apis.Convertible) error {
	switch source := obj.(type) {
	case *v1beta1.KnativeServing:
		ks.ObjectMeta = source.ObjectMeta
		ingressConfigs := convertFromIngressConfigsBeta(source)
		ks.Status = KnativeServingStatus{
			Status:    source.Status.Status,
			Version:   source.Status.Version,
			Manifests: source.Status.Manifests,
		}

		ks.Spec = KnativeServingSpec{
			ControllerCustomCerts: source.Spec.ControllerCustomCerts,
			Ingress:               ingressConfigs,
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
		return apis.ConvertFromViaProxy(ctx, source, &v1beta1.KnativeServing{}, ks)
	}
}
