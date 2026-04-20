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

package common

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	clienttesting "k8s.io/client-go/testing"
	toolscache "k8s.io/client-go/tools/cache"

	fakeciclient "sigs.k8s.io/cluster-inventory-api/client/clientset/versioned/fake"
)

// newInformerTestProvider builds a ClusterProvider backed by the given fake clientset.
func newInformerTestProvider(controllerCtx context.Context, ci *fakeciclient.Clientset) *ClusterProvider {
	return &ClusterProvider{
		entries:       map[string]*clusterEntry{},
		access:        &stubAccess{},
		ciClient:      ci,
		controllerCtx: controllerCtx,
		remoteTimeout: defaultRemoteClusterTimeout,
		clientFactory: defaultClientFactory{},
	}
}

// waitForCount polls until counter reaches want, or fails the test after 2s.
func waitForCount(t *testing.T, counter *atomic.Int32, want int32, msg string) {
	t.Helper()
	if err := wait.PollUntilContextTimeout(context.Background(), 10*time.Millisecond, 2*time.Second, true,
		func(context.Context) (bool, error) {
			return counter.Load() >= want, nil
		}); err != nil {
		t.Fatalf("%s: waited for count=%d, got=%d: %v", msg, want, counter.Load(), err)
	}
}

func TestStartInformer_Idempotent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	ci := fakeciclient.NewSimpleClientset()

	// Count List calls; a repeat StartInformer must short-circuit (no new List).
	var listCalls atomic.Int32
	ci.PrependReactor("list", "clusterprofiles", func(clienttesting.Action) (bool, runtime.Object, error) {
		listCalls.Add(1)
		return false, nil, nil
	})

	p := newInformerTestProvider(ctx, ci)

	p.StartInformer(ctx)
	firstCalls := listCalls.Load()
	if firstCalls == 0 {
		t.Fatalf("expected first StartInformer to issue at least one List, got 0")
	}
	if !p.informerStarted {
		t.Fatal("informerStarted flag not set after first StartInformer")
	}

	// A second StartInformer must be a no-op.
	p.StartInformer(ctx)

	snapshot := listCalls.Load()
	p.StartInformer(ctx)
	if after := listCalls.Load(); after-snapshot > 1 {
		t.Fatalf("second StartInformer triggered additional List calls: before=%d after=%d", snapshot, after)
	}
}

func TestStartInformer_APIUnavailable(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	ci := fakeciclient.NewSimpleClientset()
	ci.PrependReactor("list", "clusterprofiles", func(clienttesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewServiceUnavailable("injected")
	})

	p := newInformerTestProvider(ctx, ci)

	// Run in a goroutine so we can assert StartInformer returns promptly on probe failure.
	var wg sync.WaitGroup
	wg.Go(func() {
		p.StartInformer(ctx)
	})

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("StartInformer did not return after List probe failure (possible leaked goroutine)")
	}

	if !p.informerStarted {
		t.Fatal("informerStarted must be set even when the API probe fails, to prevent retry storms")
	}
}

// TestStartInformerHandlers exercises the Add/Update/Delete handlers wired by StartInformer.
func TestStartInformerHandlers(t *testing.T) {
	t.Run("Add", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)

		ci := fakeciclient.NewSimpleClientset()
		p := newInformerTestProvider(ctx, ci)

		var enqueued atomic.Int32
		p.RegisterListener(ClusterProfileListener{
			ListCRs: func(ns, name string) []types.NamespacedName {
				if ns != "fleet" || name != "worker" {
					t.Errorf("ListCRs got unexpected key %s/%s", ns, name)
				}
				return []types.NamespacedName{{Namespace: "knative-serving", Name: "default"}}
			},
			EnqueueKey: func(types.NamespacedName) {
				enqueued.Add(1)
			},
		})

		p.StartInformer(ctx)

		if _, err := ci.ApisV1alpha1().ClusterProfiles("fleet").Create(ctx,
			readyClusterProfile("fleet", "worker"), metav1.CreateOptions{}); err != nil {
			t.Fatalf("Create ClusterProfile: %v", err)
		}

		waitForCount(t, &enqueued, 1, "Add event not delivered")
	})

	t.Run("Update", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)

		existing := readyClusterProfile("fleet", "worker")
		ci := fakeciclient.NewSimpleClientset(existing)
		p := newInformerTestProvider(ctx, ci)

		var enqueued atomic.Int32
		p.RegisterListener(ClusterProfileListener{
			ListCRs: func(string, string) []types.NamespacedName {
				return []types.NamespacedName{{Namespace: "knative-serving", Name: "default"}}
			},
			EnqueueKey: func(types.NamespacedName) { enqueued.Add(1) },
		})

		p.StartInformer(ctx)

		// Wait for the initial-list Add event before mutating.
		waitForCount(t, &enqueued, 1, "initial Add event not delivered")
		before := enqueued.Load()

		updated := existing.DeepCopy()
		updated.Labels = map[string]string{"changed": "yes"}
		if _, err := ci.ApisV1alpha1().ClusterProfiles("fleet").Update(ctx,
			updated, metav1.UpdateOptions{}); err != nil {
			t.Fatalf("Update ClusterProfile: %v", err)
		}

		if err := wait.PollUntilContextTimeout(ctx, 10*time.Millisecond, 2*time.Second, true,
			func(context.Context) (bool, error) {
				return enqueued.Load() > before, nil
			}); err != nil {
			t.Fatalf("Update event not delivered: before=%d after=%d: %v",
				before, enqueued.Load(), err)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)

		existing := readyClusterProfile("fleet", "worker")
		ci := fakeciclient.NewSimpleClientset(existing)
		p := newInformerTestProvider(ctx, ci)

		// Pre-populate a cache entry so we can assert Remove clears it.
		p.entries["fleet/worker"] = newTestClusterEntry("https://cached.example.com")

		var enqueued atomic.Int32
		p.RegisterListener(ClusterProfileListener{
			ListCRs: func(string, string) []types.NamespacedName {
				return []types.NamespacedName{{Namespace: "knative-serving", Name: "default"}}
			},
			EnqueueKey: func(types.NamespacedName) { enqueued.Add(1) },
		})

		p.StartInformer(ctx)

		waitForCount(t, &enqueued, 1, "initial Add event not delivered")
		before := enqueued.Load()

		if err := ci.ApisV1alpha1().ClusterProfiles("fleet").Delete(ctx,
			"worker", metav1.DeleteOptions{}); err != nil {
			t.Fatalf("Delete ClusterProfile: %v", err)
		}

		if err := wait.PollUntilContextTimeout(ctx, 10*time.Millisecond, 2*time.Second, true,
			func(context.Context) (bool, error) {
				return enqueued.Load() > before, nil
			}); err != nil {
			t.Fatalf("Delete event not delivered: before=%d after=%d: %v",
				before, enqueued.Load(), err)
		}

		// Remove should have evicted the cache entry.
		if err := wait.PollUntilContextTimeout(ctx, 10*time.Millisecond, 2*time.Second, true,
			func(context.Context) (bool, error) {
				p.mu.RLock()
				defer p.mu.RUnlock()
				_, ok := p.entries["fleet/worker"]
				return !ok, nil
			}); err != nil {
			t.Fatalf("cache entry was not removed after Delete event: %v", err)
		}
	})

	t.Run("TombstoneFallback", func(t *testing.T) {
		// Exercise handleDelete's tombstone branch (hit on dropped watch).
		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)

		ci := fakeciclient.NewSimpleClientset()
		p := newInformerTestProvider(ctx, ci)

		// Seed a cached entry that should be removed by handleDelete.
		p.entries["fleet/worker"] = newTestClusterEntry("https://cached.example.com")

		var enqueued atomic.Int32
		p.RegisterListener(ClusterProfileListener{
			ListCRs: func(ns, name string) []types.NamespacedName {
				if ns != "fleet" || name != "worker" {
					t.Errorf("ListCRs got unexpected key %s/%s", ns, name)
				}
				return []types.NamespacedName{{Namespace: "knative-serving", Name: "default"}}
			},
			EnqueueKey: func(types.NamespacedName) { enqueued.Add(1) },
		})

		cp := readyClusterProfile("fleet", "worker")
		tombstone := toolscache.DeletedFinalStateUnknown{
			Key: "fleet/worker",
			Obj: cp,
		}
		p.handleDelete(tombstone)

		if got := enqueued.Load(); got != 1 {
			t.Fatalf("tombstone delete: EnqueueKey calls = %d, want 1", got)
		}
		p.mu.RLock()
		_, stillCached := p.entries["fleet/worker"]
		p.mu.RUnlock()
		if stillCached {
			t.Fatal("tombstone delete: cache entry was not removed")
		}

		// Bad objects (non-ClusterProfile, or tombstone wrapping one) must be no-ops.
		p.handleDelete("not-a-clusterprofile")
		p.handleDelete(toolscache.DeletedFinalStateUnknown{
			Key: "fleet/worker",
			Obj: "not-a-clusterprofile",
		})
		if got := enqueued.Load(); got != 1 {
			t.Fatalf("bad-obj delete path must not enqueue, got %d", got)
		}
	})
}

// TestStartInformer_ContextCancel asserts cancelling controllerCtx terminates the informer (no leak).
func TestStartInformer_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	ci := fakeciclient.NewSimpleClientset(readyClusterProfile("fleet", "worker"))
	p := newInformerTestProvider(ctx, ci)

	var enqueued atomic.Int32
	p.RegisterListener(ClusterProfileListener{
		ListCRs: func(string, string) []types.NamespacedName {
			return []types.NamespacedName{{Namespace: "ns", Name: "name"}}
		},
		EnqueueKey: func(types.NamespacedName) { enqueued.Add(1) },
	})

	p.StartInformer(ctx)

	// Confirm informer is live; then cancel and ensure no further events are delivered.
	waitForCount(t, &enqueued, 1, "initial Add event not delivered")
	cancel()

	settleCtx, settleCancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer settleCancel()
	<-settleCtx.Done()
	stableCount := enqueued.Load()

	_, _ = ci.ApisV1alpha1().ClusterProfiles("fleet").Create(context.Background(),
		readyClusterProfile("fleet", "second"), metav1.CreateOptions{})

	graceCtx, graceCancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer graceCancel()
	<-graceCtx.Done()
	if got := enqueued.Load(); got != stableCount {
		t.Fatalf("informer delivered events after context cancel: before=%d after=%d",
			stableCount, got)
	}
}
