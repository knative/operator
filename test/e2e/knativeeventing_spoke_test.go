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
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"

	"knative.dev/operator/pkg/apis/operator/base"
	"knative.dev/operator/pkg/apis/operator/v1beta1"
	"knative.dev/operator/test"
	"knative.dev/operator/test/client"
	"knative.dev/operator/test/resources"
)

// TestMulticlusterKnativeEventingSpokeDeployment verifies KnativeEventing is reconciled onto a spoke cluster and cleaned up on delete.
func TestMulticlusterKnativeEventingSpokeDeployment(t *testing.T) {
	ctx := t.Context()

	hub := client.Setup(t)
	spoke := client.SetupSpoke(t)

	names := test.ResourceNames{
		KnativeEventing: test.OperatorName,
		Namespace:       test.EventingOperatorNamespace,
	}

	ensureSpokeNamespace(ctx, t, spoke, names.Namespace)

	test.CleanupOnInterrupt(func() { test.TearDown(hub, names) })
	defer test.TearDown(hub, names)

	if err := createKnativeEventingWithSpokeRef(ctx, hub, names); err != nil {
		t.Fatalf("Failed to create KnativeEventing %q on hub: %v", names.KnativeEventing, err)
	}

	t.Run("hub-cr-ready", func(t *testing.T) {
		resources.AssertKEOperatorCRReadyStatus(t, hub, names)
		assertTargetClusterResolvedEventing(t.Context(), t, hub, names)
	})

	t.Run("spoke-deployments-ready", func(t *testing.T) {
		waitForSpokeEventingDeploymentsReady(t.Context(), t, hub, spoke, names)
	})

	t.Run("tls-resources-filtered-without-cert-manager", func(t *testing.T) {
		ctx := t.Context()
		assertNoCertManagerResourcesOnSpoke(ctx, t, spoke, names.Namespace)
	})

	t.Run("delete-and-cleanup-spoke", func(t *testing.T) {
		ctx := t.Context()
		if err := deleteHubKnativeEventing(ctx, hub, names); err != nil {
			t.Fatalf("Failed to delete hub KnativeEventing %q: %v", names.KnativeEventing, err)
		}
		if err := waitForSpokeEventingDeploymentsGone(ctx, t, spoke, names.Namespace); err != nil {
			t.Fatalf("Spoke deployments still present after deletion in namespace %q: %v",
				names.Namespace, err)
		}
		waitForSpokeManagedEventingServicesGone(ctx, t, spoke, names.Namespace)
		assertAnchorConfigMapGoneEventing(ctx, t, spoke, names)
	})
}

func createKnativeEventingWithSpokeRef(ctx context.Context, clients *test.Clients, names test.ResourceNames) error {
	ke := &v1beta1.KnativeEventing{
		ObjectMeta: metav1.ObjectMeta{
			Name:      names.KnativeEventing,
			Namespace: names.Namespace,
		},
		Spec: v1beta1.KnativeEventingSpec{
			CommonSpec: base.CommonSpec{
				ClusterProfileRef: &base.ClusterProfileReference{
					Name:      spokeClusterProfileRefName(),
					Namespace: spokeClusterProfileRefNamespace(),
				},
			},
		},
	}
	_, err := clients.KnativeEventing().Create(ctx, ke, metav1.CreateOptions{})
	if apierrs.IsAlreadyExists(err) {
		return nil
	}
	return err
}

func deleteHubKnativeEventing(ctx context.Context, clients *test.Clients, names test.ResourceNames) error {
	if err := clients.KnativeEventing().Delete(ctx, names.KnativeEventing, metav1.DeleteOptions{}); err != nil {
		if apierrs.IsNotFound(err) {
			return nil
		}
		return err
	}
	return wait.PollUntilContextTimeout(ctx, spokeWaitInterval, spokeGoneTimeout, true,
		func(ctx context.Context) (bool, error) {
			_, err := clients.KnativeEventing().Get(ctx, names.KnativeEventing, metav1.GetOptions{})
			if apierrs.IsNotFound(err) {
				return true, nil
			}
			return false, err
		})
}

func waitForSpokeEventingDeploymentsReady(ctx context.Context, t *testing.T, hub *test.Clients, spoke *test.Clients, names test.ResourceNames) {
	t.Helper()

	t.Logf("Waiting up to %s for hub KnativeEventing %s/%s to report %s=True",
		hubResolveTimeout, names.Namespace, names.KnativeEventing, base.TargetClusterResolved)

	var lastResolveStatus string
	resolveErr := wait.PollUntilContextTimeout(ctx, spokeWaitInterval, hubResolveTimeout, true,
		func(ctx context.Context) (bool, error) {
			ke, err := hub.KnativeEventing().Get(ctx, names.KnativeEventing, metav1.GetOptions{})
			if err != nil {
				if apierrs.IsNotFound(err) {
					return false, nil
				}
				return false, err
			}
			cond := ke.Status.GetCondition(base.TargetClusterResolved)
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
		t.Fatalf("hub KnativeEventing %s/%s did not reach %s=True: %v",
			names.Namespace, names.KnativeEventing, base.TargetClusterResolved, resolveErr)
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

func waitForSpokeEventingDeploymentsGone(ctx context.Context, t *testing.T, clients *test.Clients, namespace string) error {
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

func assertTargetClusterResolvedEventing(ctx context.Context, t *testing.T, hub *test.Clients, names test.ResourceNames) {
	t.Helper()
	ke, err := hub.KnativeEventing().Get(ctx, names.KnativeEventing, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get hub KnativeEventing %q: %v", names.KnativeEventing, err)
	}
	cond := ke.Status.GetCondition(base.TargetClusterResolved)
	if cond == nil {
		t.Fatalf("hub KnativeEventing %q missing %s condition", names.KnativeEventing, base.TargetClusterResolved)
	}
	if cond.Status != corev1.ConditionTrue {
		t.Fatalf("hub KnativeEventing %q %s = %s, want True (reason=%q message=%q)",
			names.KnativeEventing, base.TargetClusterResolved, cond.Status, cond.Reason, cond.Message)
	}
}

func waitForSpokeManagedEventingServicesGone(ctx context.Context, t *testing.T, spoke *test.Clients, namespace string) {
	t.Helper()
	const managedBySelector = "app.kubernetes.io/name=knative-eventing"
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

func assertAnchorConfigMapGoneEventing(ctx context.Context, t *testing.T, spoke *test.Clients, names test.ResourceNames) {
	t.Helper()
	anchorName := "knativeeventing-" + names.KnativeEventing + "-root-owner"
	_, err := spoke.KubeClient.CoreV1().ConfigMaps(names.Namespace).Get(ctx, anchorName, metav1.GetOptions{})
	if err == nil {
		t.Fatalf("Anchor ConfigMap %q still exists in spoke namespace %q", anchorName, names.Namespace)
	}
	if !apierrs.IsNotFound(err) {
		t.Fatalf("Unexpected error checking anchor ConfigMap %q: %v", anchorName, err)
	}
}

// assertNoCertManagerResourcesOnSpoke fails if any cert-manager CRs exist in the spoke eventing namespace.
// Missing CRDs are treated as success.
func assertNoCertManagerResourcesOnSpoke(ctx context.Context, t *testing.T, spoke *test.Clients, namespace string) {
	t.Helper()
	gvrs := []schema.GroupVersionResource{
		{Group: "cert-manager.io", Version: "v1", Resource: "certificates"},
		{Group: "cert-manager.io", Version: "v1", Resource: "issuers"},
		{Group: "cert-manager.io", Version: "v1", Resource: "clusterissuers"},
		{Group: "trust.cert-manager.io", Version: "v1alpha1", Resource: "bundles"},
	}
	for _, gvr := range gvrs {
		list, err := spoke.Dynamic.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			// CRD not installed on spoke => predicate trivially satisfied.
			if apierrs.IsNotFound(err) || isNoMatchErr(err) {
				t.Logf("cert-manager resource %s not registered on spoke (expected): %v", gvr.String(), err)
				continue
			}
			t.Fatalf("unexpected error listing %s on spoke: %v", gvr.String(), err)
		}
		if len(list.Items) != 0 {
			names := make([]string, 0, len(list.Items))
			for _, it := range list.Items {
				names = append(names, it.GetName())
			}
			t.Fatalf("unexpected %s resources present on spoke ns %q: %v", gvr.String(), namespace, names)
		}
	}
}

// isNoMatchErr reports whether err indicates the server has no matching resource type (CRD absent).
func isNoMatchErr(err error) bool {
	if err == nil {
		return false
	}
	if meta.IsNoMatchError(err) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "no matches for kind") ||
		strings.Contains(msg, "could not find the requested resource") ||
		strings.Contains(msg, "the server could not find the requested resource")
}
