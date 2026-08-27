//go:build e2e && multicluster
// +build e2e,multicluster

/*
Copyright 2026 The Knative Authors

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

package e2e

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"knative.dev/operator/test"
)

func waitForSpokeDeploymentsAvailable(ctx context.Context, t *testing.T, spoke *test.Clients, namespace string) {
	t.Helper()
	t.Logf("Waiting up to %s for all Deployments in spoke namespace %q to become Available",
		spokeReadyTimeout, namespace)

	var (
		lastTotal    = -1
		lastReady    = -1
		lastObserved []appsv1.Deployment
	)
	pollErr := wait.PollUntilContextTimeout(ctx, spokeWaitInterval, spokeReadyTimeout, true,
		func(ctx context.Context) (bool, error) {
			dpList, err := spoke.KubeClient.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
			if err != nil {
				return false, err
			}
			lastObserved = dpList.Items
			total := len(dpList.Items)
			ready := 0
			for _, deployment := range dpList.Items {
				if isDeploymentAvailable(&deployment) {
					ready++
				}
			}
			if total != lastTotal || ready != lastReady {
				t.Logf("spoke ns %q: %d/%d Deployments Available", namespace, ready, total)
				lastTotal = total
				lastReady = ready
			}
			if total == 0 {
				return false, nil
			}
			return ready == total, nil
		})
	if pollErr != nil {
		t.Logf("Spoke deployments did not become ready in namespace %q. Last observed state:", namespace)
		dumpDeployments(t, lastObserved)
		t.Fatalf("Spoke deployments did not become ready in namespace %q: %v", namespace, pollErr)
	}
}

func waitForSpokeDeploymentsGone(ctx context.Context, t *testing.T, clients *test.Clients, namespace string) error {
	t.Helper()
	t.Logf("Waiting up to %s for all Deployments in spoke namespace %q to disappear",
		spokeGoneTimeout, namespace)

	lastCount := -1
	return wait.PollUntilContextTimeout(ctx, spokeWaitInterval, spokeGoneTimeout, true,
		func(ctx context.Context) (bool, error) {
			dpList, err := clients.KubeClient.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
			if err != nil {
				if apierrs.IsNotFound(err) {
					return true, nil
				}
				return false, err
			}
			if len(dpList.Items) != lastCount {
				t.Logf("spoke ns %q: %d Deployments remaining", namespace, len(dpList.Items))
				lastCount = len(dpList.Items)
			}
			return len(dpList.Items) == 0, nil
		})
}

func isDeploymentAvailable(deployment *appsv1.Deployment) bool {
	for _, condition := range deployment.Status.Conditions {
		if condition.Type == appsv1.DeploymentAvailable && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func dumpDeployments(t *testing.T, items []appsv1.Deployment) {
	t.Helper()
	if len(items) == 0 {
		t.Logf("  (no deployments observed)")
		return
	}
	names := make([]string, 0, len(items))
	byName := make(map[string]appsv1.Deployment, len(items))
	for _, deployment := range items {
		names = append(names, deployment.Name)
		byName[deployment.Name] = deployment
	}
	sort.Strings(names)
	for _, name := range names {
		deployment := byName[name]
		conditions := make([]string, 0, len(deployment.Status.Conditions))
		for _, condition := range deployment.Status.Conditions {
			conditions = append(conditions, fmt.Sprintf("%s=%s(%s)", condition.Type, condition.Status, condition.Reason))
		}
		t.Logf("  - %s: replicas=%d/%d ready=%d available=%d updated=%d conditions=[%s]",
			name,
			deployment.Status.ReadyReplicas, deployment.Status.Replicas,
			deployment.Status.ReadyReplicas, deployment.Status.AvailableReplicas, deployment.Status.UpdatedReplicas,
			strings.Join(conditions, ","))
	}
}

func waitForSpokeManagedServicesGone(
	ctx context.Context,
	t *testing.T,
	spoke *test.Clients,
	namespace string,
	managedBySelector string,
) {
	t.Helper()
	t.Logf("Waiting up to %s for operator-managed Services (%s) in spoke namespace %q to disappear",
		spokeGoneTimeout, managedBySelector, namespace)

	lastCount := -1
	err := wait.PollUntilContextTimeout(ctx, spokeWaitInterval, spokeGoneTimeout, true,
		func(ctx context.Context) (bool, error) {
			svcList, err := spoke.KubeClient.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{
				LabelSelector: managedBySelector,
			})
			if err != nil {
				if apierrs.IsNotFound(err) {
					return true, nil
				}
				return false, err
			}
			if len(svcList.Items) != lastCount {
				t.Logf("spoke ns %q: %d operator-managed Services remaining", namespace, len(svcList.Items))
				lastCount = len(svcList.Items)
			}
			return len(svcList.Items) == 0, nil
		})
	if err != nil {
		t.Fatalf("Spoke namespace %q still has operator-managed Services after cleanup: %v", namespace, err)
	}
}

func assertAnchorConfigMapGone(
	ctx context.Context,
	t *testing.T,
	spoke *test.Clients,
	namespace string,
	anchorName string,
) {
	t.Helper()
	_, err := spoke.KubeClient.CoreV1().ConfigMaps(namespace).Get(ctx, anchorName, metav1.GetOptions{})
	if err == nil {
		t.Fatalf("Anchor ConfigMap %q still exists in spoke namespace %q", anchorName, namespace)
	}
	if !apierrs.IsNotFound(err) {
		t.Fatalf("Unexpected error checking anchor ConfigMap %q: %v", anchorName, err)
	}
}
