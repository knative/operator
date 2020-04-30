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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
	"knative.dev/pkg/logging"
)

func CheckDeployments(ctx context.Context, kubeClient kubernetes.Interface, manifest *mf.Manifest, status v1alpha1.KComponentStatus) error {
	logger := logging.FromContext(ctx)

	logger.Debug("Checking deployments")
	available := func(d *appsv1.Deployment) bool {
		for _, c := range d.Status.Conditions {
			if c.Type == appsv1.DeploymentAvailable && c.Status == corev1.ConditionTrue {
				return true
			}
		}
		return false
	}
	for _, u := range manifest.Filter(mf.ByKind("Deployment")).Resources() {
		deployment, err := kubeClient.AppsV1().Deployments(u.GetNamespace()).Get(u.GetName(), metav1.GetOptions{})
		if err != nil {
			status.MarkDeploymentsNotReady()
			if errors.IsNotFound(err) {
				return nil
			}
			return err
		}
		if !available(deployment) {
			status.MarkDeploymentsNotReady()
			return nil
		}
	}
	status.MarkDeploymentsAvailable()
	return nil
}
