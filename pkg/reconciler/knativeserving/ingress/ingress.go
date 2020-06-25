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

package ingress

import (
	"context"

	mf "github.com/manifestival/manifestival"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
)

const providerLabel = "networking.knative.dev/ingress-provider"

func ingressFilter(name string) mf.Predicate {
	return func(u *unstructured.Unstructured) bool {
		provider, hasLabel := u.GetLabels()[providerLabel]
		if !hasLabel {
			return true
		}

		return provider == name
	}
}

func Filters(ks *v1alpha1.KnativeServing) mf.Predicate {
	if ks.Spec.Ingress == nil {
		return mf.Any(istioFilter)
	}

	var filters []mf.Predicate
	if ks.Spec.Ingress.Istio.Enabled {
		filters = append(filters, istioFilter)
	}
	if ks.Spec.Ingress.Kourier.Enabled {
		filters = append(filters, kourierFilter)
	}
	return mf.Any(filters...)
}

func Transformers(ctx context.Context, ks *v1alpha1.KnativeServing) []mf.Transformer {
	if ks.Spec.Ingress == nil {
		return istioTransformers(ctx, ks)
	}
	var transformers []mf.Transformer
	if ks.Spec.Ingress.Istio.Enabled {
		transformers = append(transformers, istioTransformers(ctx, ks)...)
	}
	if ks.Spec.Ingress.Kourier.Enabled {
		transformers = append(transformers, kourierTransformers(ctx, ks)...)
	}
	return transformers
}
