/*
Copyright 2024 The Knative Authors

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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"

	"knative.dev/operator/pkg/apis/operator/base"
)

// NamespaceConfigurationTransform mutates the only namespace available for knative serving or eventing
// by changing the labels and annotations.
func NamespaceConfigurationTransform(namespaceConfiguration *base.NamespaceConfiguration) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if u.GetKind() != "Namespace" || namespaceConfiguration == nil {
			return nil
		}
		namespace := &corev1.Namespace{}
		err := scheme.Scheme.Convert(u, namespace, nil)
		if err != nil {
			return err
		}

		// Override the labels for the namespace
		if namespace.GetLabels() == nil {
			namespace.Labels = map[string]string{}
		}

		for key, val := range namespaceConfiguration.Labels {
			namespace.Labels[key] = val
		}

		// Override the annotations for the namespace
		if namespace.GetAnnotations() == nil {
			namespace.Annotations = map[string]string{}
		}

		for key, val := range namespaceConfiguration.Annotations {
			namespace.Annotations[key] = val
		}

		err = scheme.Scheme.Convert(namespace, u, nil)
		if err != nil {
			return err
		}
		return nil
	}
}
