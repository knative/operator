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

	mf "github.com/manifestival/manifestival"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"knative.dev/pkg/logging"

	"knative.dev/operator/pkg/apis/operator/base"
)

// transformers that are common to all components.
func transformers(ctx context.Context, obj base.KComponent) []mf.Transformer {
	logger := logging.FromContext(ctx)
	return []mf.Transformer{
		injectOwner(obj),
		mf.InjectNamespace(obj.GetNamespace()),
		NamespaceConfigurationTransform(obj.GetSpec().GetNamespaceConfiguration()),
		HighAvailabilityTransform(obj),
		ImageTransform(obj.GetSpec().GetRegistry(), logger),
		JobTransform(obj),
		ConfigMapTransform(obj.GetSpec().GetConfig(), logger),
		ResourceRequirementsTransform(obj, logger),
		OverridesTransform(obj.GetSpec().GetWorkloadOverrides(), logger),
		ServicesTransform(obj, logger),
		PodDisruptionBudgetsTransform(obj, logger),
	}
}

func injectOwner(owner mf.Owner) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if u.GetNamespace() != "" {
			u.SetOwnerReferences([]v1.OwnerReference{*v1.NewControllerRef(owner, owner.GroupVersionKind())})
		}
		return nil
	}
}

// Transform will mutate the passed-by-reference manifest with one
// transformed by platform, common, and any extra passed in
func Transform(ctx context.Context, manifest *mf.Manifest, instance base.KComponent, extra ...mf.Transformer) error {
	logger := logging.FromContext(ctx)
	logger.Debug("Transforming manifest")

	transformers := transformers(ctx, instance)
	transformers = append(transformers, extra...)

	m, err := manifest.Transform(transformers...)
	if err != nil {
		instance.GetStatus().MarkInstallFailed(err.Error())
		return err
	}
	*manifest = m
	return nil
}

// InjectNamespace will mutate the namespace of all installed resources
func InjectNamespace(manifest *mf.Manifest, instance base.KComponent, extra ...mf.Transformer) error {
	transformers := []mf.Transformer{
		mf.InjectNamespace(instance.GetNamespace()),
	}
	transformers = append(transformers, extra...)
	m, err := manifest.Transform(transformers...)
	if err != nil {
		instance.GetStatus().MarkInstallFailed(err.Error())
		return err
	}
	*manifest = m
	return nil
}

// InjectLabel adds the given key and value as label.
func InjectLabel(key, value string) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		curr := u.GetLabels()
		if curr == nil {
			curr = map[string]string{}
		}
		curr[key] = value
		u.SetLabels(curr)
		return nil
	}
}
