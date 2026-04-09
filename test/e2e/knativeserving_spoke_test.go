//go:build e2e && multicluster
// +build e2e,multicluster

/*
Copyright 2025 The Knative Authors

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
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"knative.dev/operator/pkg/apis/operator/base"
	"knative.dev/operator/pkg/apis/operator/v1beta1"
	"knative.dev/operator/test"
	"knative.dev/operator/test/client"
	"knative.dev/operator/test/resources"
)

const (
	defaultSpokeClusterProfileName      = "spoke"
	defaultSpokeClusterProfileNamespace = "default"

	spokeWaitInterval = 5 * time.Second
	spokeReadyTimeout = 5 * time.Minute
	spokeGoneTimeout  = 3 * time.Minute
	hubResolveTimeout = 60 * time.Second
)

func spokeClusterProfileRefName() string {
	if v := os.Getenv("SPOKE_CLUSTER_NAME"); v != "" {
		return v
	}
	return defaultSpokeClusterProfileName
}

func spokeClusterProfileRefNamespace() string {
	if v := os.Getenv("SPOKE_CLUSTER_NAMESPACE"); v != "" {
		return v
	}
	return defaultSpokeClusterProfileNamespace
}

func TestMulticlusterKnativeServingSpokeDeployment(t *testing.T) {
	ctx := t.Context()

	hub := client.Setup(t)
	spoke := client.SetupSpoke(t)

	names := test.ResourceNames{
		KnativeServing: test.OperatorName,
		Namespace:      test.ServingOperatorNamespace,
	}

	ensureSpokeNamespace(ctx, t, spoke, names.Namespace)

	test.CleanupOnInterrupt(func() { test.TearDown(hub, names) })
	defer test.TearDown(hub, names)

	if err := createKnativeServingWithSpokeRef(ctx, hub, names); err != nil {
		t.Fatalf("Failed to create KnativeServing %q on hub: %v", names.KnativeServing, err)
	}

	t.Run("hub-cr-ready", func(t *testing.T) {
		resources.AssertKSOperatorCRReadyStatus(t, hub, names)
		assertTargetClusterResolved(t.Context(), t, hub, names)
	})

	t.Run("spoke-deployments-ready", func(t *testing.T) {
		waitForSpokeDeploymentsReady(t.Context(), t, hub, spoke, names)
	})

	t.Run("delete-and-cleanup-spoke", func(t *testing.T) {
		ctx := t.Context()
		if err := deleteHubKnativeServing(ctx, hub, names); err != nil {
			t.Fatalf("Failed to delete hub KnativeServing %q: %v", names.KnativeServing, err)
		}
		if err := waitForSpokeDeploymentsGone(ctx, t, spoke, names.Namespace); err != nil {
			t.Fatalf("Spoke deployments still present after deletion in namespace %q: %v",
				names.Namespace, err)
		}
		waitForSpokeManagedServicesGone(ctx, t, spoke, names.Namespace)
		assertAnchorConfigMapGone(ctx, t, spoke, names)
	})
}

func createKnativeServingWithSpokeRef(ctx context.Context, clients *test.Clients, names test.ResourceNames) error {
	ks := &v1beta1.KnativeServing{
		ObjectMeta: metav1.ObjectMeta{
			Name:      names.KnativeServing,
			Namespace: names.Namespace,
		},
		Spec: v1beta1.KnativeServingSpec{
			CommonSpec: base.CommonSpec{
				ClusterProfileRef: &base.ClusterProfileReference{
					Name:      spokeClusterProfileRefName(),
					Namespace: spokeClusterProfileRefNamespace(),
				},
				Config: map[string]map[string]string{
					"network": {
						"ingress-class": "gateway-api.ingress.networking.knative.dev",
					},
				},
			},
			Ingress: &v1beta1.IngressConfigs{
				Istio:      base.IstioIngressConfiguration{Enabled: false},
				GatewayAPI: base.GatewayAPIIngressConfiguration{Enabled: true},
			},
		},
	}
	_, err := clients.KnativeServing().Create(ctx, ks, metav1.CreateOptions{})
	if apierrs.IsAlreadyExists(err) {
		return nil
	}
	return err
}

func deleteHubKnativeServing(ctx context.Context, clients *test.Clients, names test.ResourceNames) error {
	if err := clients.KnativeServing().Delete(ctx, names.KnativeServing, metav1.DeleteOptions{}); err != nil {
		if apierrs.IsNotFound(err) {
			return nil
		}
		return err
	}
	return wait.PollUntilContextTimeout(ctx, spokeWaitInterval, spokeGoneTimeout, true,
		func(ctx context.Context) (bool, error) {
			_, err := clients.KnativeServing().Get(ctx, names.KnativeServing, metav1.GetOptions{})
			if apierrs.IsNotFound(err) {
				return true, nil
			}
			return false, err
		})
}

func ensureSpokeNamespace(ctx context.Context, t *testing.T, clients *test.Clients, namespace string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	_, err := clients.KubeClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: namespace},
	}, metav1.CreateOptions{})
	if err != nil && !apierrs.IsAlreadyExists(err) {
		t.Fatalf("Failed to ensure spoke namespace %q: %v", namespace, err)
	}
}

func waitForSpokeDeploymentsReady(ctx context.Context, t *testing.T, hub *test.Clients, spoke *test.Clients, names test.ResourceNames) {
	t.Helper()

	t.Logf("Waiting up to %s for hub KnativeServing %s/%s to report %s=True",
		hubResolveTimeout, names.Namespace, names.KnativeServing, base.TargetClusterResolved)

	var lastResolveStatus string
	resolveErr := wait.PollUntilContextTimeout(ctx, spokeWaitInterval, hubResolveTimeout, true,
		func(ctx context.Context) (bool, error) {
			ks, err := hub.KnativeServing().Get(ctx, names.KnativeServing, metav1.GetOptions{})
			if err != nil {
				if apierrs.IsNotFound(err) {
					return false, nil
				}
				return false, err
			}
			cond := ks.Status.GetCondition(base.TargetClusterResolved)
			if cond == nil {
				if lastResolveStatus != "Unknown(missing)" {
					t.Logf("hub CR %s condition not yet set", base.TargetClusterResolved)
					lastResolveStatus = "Unknown(missing)"
				}
				return false, nil
			}
			status := fmt.Sprintf("%s/%s/%s", cond.Status, cond.Reason, cond.Message)
			if status != lastResolveStatus {
				t.Logf("hub CR %s=%s reason=%q message=%q",
					base.TargetClusterResolved, cond.Status, cond.Reason, cond.Message)
				lastResolveStatus = status
			}
			switch cond.Status {
			case corev1.ConditionTrue:
				return true, nil
			case corev1.ConditionFalse:
				return false, fmt.Errorf("hub CR %s=False reason=%q message=%q",
					base.TargetClusterResolved, cond.Reason, cond.Message)
			default:
				return false, nil
			}
		})
	if resolveErr != nil {
		t.Fatalf("hub KnativeServing %s/%s did not reach %s=True: %v",
			names.Namespace, names.KnativeServing, base.TargetClusterResolved, resolveErr)
	}

	t.Logf("Waiting up to %s for all Deployments in spoke namespace %q to become Available",
		spokeReadyTimeout, names.Namespace)

	var (
		lastTotal    = -1
		lastReady    = -1
		lastObserved []appsv1.Deployment
	)
	pollErr := wait.PollUntilContextTimeout(ctx, spokeWaitInterval, spokeReadyTimeout, true,
		func(ctx context.Context) (bool, error) {
			dpList, err := spoke.KubeClient.AppsV1().Deployments(names.Namespace).List(ctx, metav1.ListOptions{})
			if err != nil {
				return false, err
			}
			lastObserved = dpList.Items
			total := len(dpList.Items)
			ready := 0
			for _, d := range dpList.Items {
				if isDeploymentAvailable(&d) {
					ready++
				}
			}
			if total != lastTotal || ready != lastReady {
				t.Logf("spoke ns %q: %d/%d Deployments Available", names.Namespace, ready, total)
				lastTotal = total
				lastReady = ready
			}
			if total == 0 {
				return false, nil
			}
			return ready == total, nil
		})
	if pollErr != nil {
		t.Logf("Spoke deployments did not become ready in namespace %q. Last observed state:",
			names.Namespace)
		dumpDeployments(t, lastObserved)
		t.Fatalf("Spoke deployments did not become ready in namespace %q: %v",
			names.Namespace, pollErr)
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

func isDeploymentAvailable(d *appsv1.Deployment) bool {
	for _, c := range d.Status.Conditions {
		if c.Type == appsv1.DeploymentAvailable && c.Status == corev1.ConditionTrue {
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
	for _, d := range items {
		names = append(names, d.Name)
		byName[d.Name] = d
	}
	sort.Strings(names)
	for _, n := range names {
		d := byName[n]
		conds := make([]string, 0, len(d.Status.Conditions))
		for _, c := range d.Status.Conditions {
			conds = append(conds, fmt.Sprintf("%s=%s(%s)", c.Type, c.Status, c.Reason))
		}
		t.Logf("  - %s: replicas=%d/%d ready=%d available=%d updated=%d conditions=[%s]",
			n,
			d.Status.ReadyReplicas, d.Status.Replicas,
			d.Status.ReadyReplicas, d.Status.AvailableReplicas, d.Status.UpdatedReplicas,
			strings.Join(conds, ","))
	}
}

func assertTargetClusterResolved(ctx context.Context, t *testing.T, hub *test.Clients, names test.ResourceNames) {
	t.Helper()
	ks, err := hub.KnativeServing().Get(ctx, names.KnativeServing, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get hub KnativeServing %q: %v", names.KnativeServing, err)
	}
	cond := ks.Status.GetCondition(base.TargetClusterResolved)
	if cond == nil {
		t.Fatalf("hub KnativeServing %q missing %s condition", names.KnativeServing, base.TargetClusterResolved)
	}
	if cond.Status != corev1.ConditionTrue {
		t.Fatalf("hub KnativeServing %q %s = %s, want True (reason=%q message=%q)",
			names.KnativeServing, base.TargetClusterResolved, cond.Status, cond.Reason, cond.Message)
	}
}

func waitForSpokeManagedServicesGone(ctx context.Context, t *testing.T, spoke *test.Clients, namespace string) {
	t.Helper()
	const managedBySelector = "app.kubernetes.io/name=knative-serving"
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

func assertAnchorConfigMapGone(ctx context.Context, t *testing.T, spoke *test.Clients, names test.ResourceNames) {
	t.Helper()
	anchorName := "knativeserving-" + names.KnativeServing + "-root-owner"
	_, err := spoke.KubeClient.CoreV1().ConfigMaps(names.Namespace).Get(ctx, anchorName, metav1.GetOptions{})
	if err == nil {
		t.Fatalf("Anchor ConfigMap %q still exists in spoke namespace %q", anchorName, names.Namespace)
	}
	if !apierrs.IsNotFound(err) {
		t.Fatalf("Unexpected error checking anchor ConfigMap %q: %v", anchorName, err)
	}
}
