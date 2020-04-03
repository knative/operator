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
	mf "github.com/manifestival/manifestival"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"

	eventingv1alpha1 "knative.dev/operator/pkg/apis/eventing/v1alpha1"
)

var log = zap.NewExample().Sugar()

type Platforms []func(kubernetes.Interface, *zap.SugaredLogger) (mf.Transformer, error)

func (platforms Platforms) Transformers(kubeClientSet kubernetes.Interface, instance *eventingv1alpha1.KnativeEventing, slog *zap.SugaredLogger) ([]mf.Transformer, error) {
	log = slog.Named("extensions")
	result := []mf.Transformer{
		mf.InjectOwner(instance),
		mf.InjectNamespace(instance.GetNamespace()),
		DeploymentTransform(instance, log),
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
