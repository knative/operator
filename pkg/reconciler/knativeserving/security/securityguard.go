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

package security

import (
	"context"

	mf "github.com/manifestival/manifestival"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
	"knative.dev/pkg/logging"

	"knative.dev/operator/pkg/apis/operator/v1beta1"
)

var (
	// SecurityGuardVersion is the hash map to maintain the relationship between knative version and the security guard version
	SecurityGuardVersion = map[string]string{
		"v1.9": "0.5",
		"v1.8": "0.5",
	}

	// QueueProxyMountPodInfoKey is the key for the QueueProxyMountPodInfo
	QueueProxyMountPodInfoKey = "queueproxy.mount-podinfo"
)

func securityGuardTransformers(ctx context.Context, instance *v1beta1.KnativeServing) []mf.Transformer {
	logger := logging.FromContext(ctx)
	return []mf.Transformer{configMapTransform(instance, logger)}
}

func configMapTransform(instance *v1beta1.KnativeServing, log *zap.SugaredLogger) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if u.GetKind() == "ConfigMap" && u.GetName() == "config-features" {
			if instance.Spec.Security == nil || !instance.Spec.Security.SecurityGuard.Enabled {
				return nil
			}
			var configMap = &corev1.ConfigMap{}
			err := scheme.Scheme.Convert(u, configMap, nil)
			if err != nil {
				log.Error(err, "Error converting Unstructured to ConfigMap", "unstructured", u, "configMap", configMap)
				return err
			}

			// Set the value allowed to QueueProxyMountPodInfoKey
			if configMap.Data == nil {
				configMap.Data = map[string]string{}
			}
			configMap.Data[QueueProxyMountPodInfoKey] = "allowed"

			err = scheme.Scheme.Convert(configMap, u, nil)
			if err != nil {
				return err
			}
			// The zero-value timestamp defaulted by the conversion causes
			// superfluous updates
			u.SetCreationTimestamp(metav1.Time{})
		}
		return nil
	}
}
