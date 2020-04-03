/*
Copyright 2019 The Knative Authors

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

	mf "github.com/manifestival/manifestival"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	servingv1alpha1 "knative.dev/operator/pkg/apis/serving/v1alpha1"
)

var log = zap.NewExample().Sugar()

type Platforms []func(kubernetes.Interface, *zap.SugaredLogger) (mf.Transformer, error)

// pfKey is used as the key for associating Platforms with the context.
type pfKey struct{}

func (platforms Platforms) Transformers(kubeClientSet kubernetes.Interface, instance *servingv1alpha1.KnativeServing, slog *zap.SugaredLogger) ([]mf.Transformer, error) {
	log = slog.Named("extensions")
	result := []mf.Transformer{
		mf.InjectOwner(instance),
		mf.InjectNamespace(instance.GetNamespace()),
		ConfigMapTransform(instance, log),
		ImageTransform(instance, log),
		GatewayTransform(instance, log),
		CustomCertsTransform(instance, log),
		HighAvailabilityTransform(instance, log),
		ResourceRequirementsTransform(instance, log),
	}
	for _, fn := range platforms {
		transformer, err := fn(kubeClientSet, log)
		if err != nil {
			return result, err
		}
		if transformer != nil {
			result = append(result, transformer)
		}
	}
	return result, nil
}

// WithPlatforms attaches the given Platforms to the provided context.
func WithPlatforms(ctx context.Context, pf Platforms) context.Context {
	return context.WithValue(ctx, pfKey{}, pf)
}

// GetPlatforms extracts the Platforms from the context.
func GetPlatforms(ctx context.Context) Platforms {
	untyped := ctx.Value(pfKey{})
	if untyped == nil {
		return nil
	}
	return untyped.(Platforms)
}
