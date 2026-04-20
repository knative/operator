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
	"errors"
	"strings"
	"sync"
	"testing"

	mf "github.com/manifestival/manifestival"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"

	"knative.dev/operator/pkg/apis/operator/base"
	"knative.dev/operator/pkg/apis/operator/v1beta1"

	clusterinventoryv1alpha1 "sigs.k8s.io/cluster-inventory-api/apis/v1alpha1"
	fakeciclient "sigs.k8s.io/cluster-inventory-api/client/clientset/versioned/fake"
)

type stubAccess struct {
	mu         sync.Mutex
	buildCount int
	buildFn    func(*clusterinventoryv1alpha1.ClusterProfile) (*rest.Config, error)
}

func (s *stubAccess) BuildConfigFromCP(cp *clusterinventoryv1alpha1.ClusterProfile) (*rest.Config, error) {
	s.mu.Lock()
	s.buildCount++
	s.mu.Unlock()
	if s.buildFn == nil {
		return &rest.Config{Host: "https://stub.example.com"}, nil
	}
	return s.buildFn(cp)
}

func (s *stubAccess) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buildCount
}

func readyClusterProfile(namespace, name string) *clusterinventoryv1alpha1.ClusterProfile {
	return &clusterinventoryv1alpha1.ClusterProfile{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Status: clusterinventoryv1alpha1.ClusterProfileStatus{
			Conditions: []metav1.Condition{
				{
					Type:               clusterinventoryv1alpha1.ClusterConditionControlPlaneHealthy,
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.Now(),
					Reason:             "Ready",
					Message:            "healthy",
				},
			},
		},
	}
}

func newTestProviderWithStubAccess(stub *stubAccess, cps ...*clusterinventoryv1alpha1.ClusterProfile) *ClusterProvider {
	objs := make([]runtime.Object, 0, len(cps))
	for _, cp := range cps {
		objs = append(objs, cp)
	}
	ci := fakeciclient.NewSimpleClientset(objs...)
	return &ClusterProvider{
		entries:       map[string]*clusterEntry{},
		access:        stub,
		ciClient:      ci,
		controllerCtx: context.Background(),
		remoteTimeout: defaultRemoteClusterTimeout,
		clientFactory: defaultClientFactory{},
	}
}

func newTestClusterEntry(host string) *clusterEntry {
	ctx, cancel := context.WithCancel(context.Background())
	return &clusterEntry{
		restConfig: &rest.Config{Host: host},
		cancel:     cancel,
		ctx:        ctx,
	}
}

func TestClusterProvider_Refresh_AccessFailure(t *testing.T) {
	stub := &stubAccess{
		buildFn: func(*clusterinventoryv1alpha1.ClusterProfile) (*rest.Config, error) {
			return nil, errors.New("simulated exec-plugin failure")
		},
	}
	provider := newTestProviderWithStubAccess(stub, readyClusterProfile("fleet", "worker"))

	if _, err := provider.Refresh(context.Background(), "fleet", "worker"); err == nil {
		t.Fatal("Refresh() = nil, want error")
	}
	if got := stub.count(); got != 1 {
		t.Fatalf("BuildConfigFromCP call count = %d, want 1", got)
	}
	if _, _, err := provider.Get(context.Background(), "fleet/worker"); !errors.Is(err, errClusterNotResolved) {
		t.Fatalf("Get() = %v, want errClusterNotResolved", err)
	}
}

func TestClusterProvider_Refresh_NoOpAccessReturnsDisabledError(t *testing.T) {
	ci := fakeciclient.NewSimpleClientset(readyClusterProfile("fleet", "worker"))
	provider := &ClusterProvider{
		entries:       map[string]*clusterEntry{},
		access:        NoOpClusterProfileAccess{},
		ciClient:      ci,
		controllerCtx: context.Background(),
		remoteTimeout: defaultRemoteClusterTimeout,
	}
	_, err := provider.Refresh(context.Background(), "fleet", "worker")
	if !errors.Is(err, errMulticlusterDisabled) {
		t.Fatalf("Refresh() error = %v, want wrapping errMulticlusterDisabled", err)
	}
}

func TestClusterProvider_GetOrRefresh_CacheHit(t *testing.T) {
	stub := &stubAccess{}
	provider := newTestProviderWithStubAccess(stub)

	entry := newTestClusterEntry("https://cached.example.com")
	provider.entries["fleet/worker"] = entry

	got, _, err := provider.GetOrRefresh(context.Background(), "fleet", "worker")
	if err != nil {
		t.Fatalf("GetOrRefresh() error = %v, want nil", err)
	}
	if got.RestConfig().Host != "https://cached.example.com" {
		t.Fatalf("Got unexpected entry (Host=%q)", got.RestConfig().Host)
	}
	if stub.count() != 0 {
		t.Fatalf("Cache hit must not invoke BuildConfigFromCP, got %d calls", stub.count())
	}
}

func TestClusterProvider_GetOrRefresh_CacheMiss_RefreshSucceeds(t *testing.T) {
	stub := &stubAccess{
		buildFn: func(*clusterinventoryv1alpha1.ClusterProfile) (*rest.Config, error) {
			return &rest.Config{Host: "https://fresh.example.com"}, nil
		},
	}
	provider := newTestProviderWithStubAccess(stub, readyClusterProfile("fleet", "worker"))

	got, _, err := provider.GetOrRefresh(context.Background(), "fleet", "worker")
	if err != nil {
		t.Fatalf("GetOrRefresh() error = %v, want nil", err)
	}
	if got == nil || got.RestConfig() == nil {
		t.Fatalf("Got nil entry or nil RestConfig")
	}
	if got.RestConfig().Host != "https://fresh.example.com" {
		t.Fatalf("Host = %q, want https://fresh.example.com", got.RestConfig().Host)
	}
	if got := stub.count(); got != 1 {
		t.Fatalf("BuildConfigFromCP call count = %d, want 1", got)
	}

	// Second call should hit cache.
	if _, _, err := provider.GetOrRefresh(context.Background(), "fleet", "worker"); err != nil {
		t.Fatalf("second GetOrRefresh() error = %v", err)
	}
	if stub.count() != 1 {
		t.Fatalf("Second call should hit cache; BuildConfigFromCP called %d times", stub.count())
	}
}

func TestClusterProvider_GetOrRefresh_CacheMiss_RefreshFails(t *testing.T) {
	stub := &stubAccess{
		buildFn: func(*clusterinventoryv1alpha1.ClusterProfile) (*rest.Config, error) {
			return nil, errors.New("simulated access failure")
		},
	}
	provider := newTestProviderWithStubAccess(stub, readyClusterProfile("fleet", "worker"))

	_, _, err := provider.GetOrRefresh(context.Background(), "fleet", "worker")
	if err == nil {
		t.Fatal("GetOrRefresh() = nil, want error")
	}
	if got := stub.count(); got != 1 {
		t.Fatalf("BuildConfigFromCP call count = %d, want 1", got)
	}

	// On a second call the provider should try to refresh again (no caching of errors).
	_, _, err = provider.GetOrRefresh(context.Background(), "fleet", "worker")
	if err == nil {
		t.Fatal("second GetOrRefresh() = nil, want error")
	}
	if got := stub.count(); got != 2 {
		t.Fatalf("BuildConfigFromCP call count = %d, want 2", got)
	}
}

func TestNotifyListeners_DoesNotCallRefresh(t *testing.T) {
	stub := &stubAccess{}
	provider := newTestProviderWithStubAccess(stub, readyClusterProfile("fleet", "worker"))

	var notified int
	provider.RegisterListener(ClusterProfileListener{
		ListCRs: func(ns, name string) []types.NamespacedName {
			if ns != "fleet" || name != "worker" {
				t.Errorf("Listener called with unexpected key %s/%s", ns, name)
			}
			notified++
			return []types.NamespacedName{{Namespace: "knative-serving", Name: "default"}}
		},
		EnqueueKey: func(types.NamespacedName) {},
	})

	provider.notifyListeners("fleet", "worker")

	if got := stub.count(); got != 0 {
		t.Fatalf("BuildConfigFromCP call count = %d, want 0", got)
	}
	if notified != 1 {
		t.Fatalf("listener invocation count = %d, want 1", notified)
	}
}

func assertTargetClusterNotResolved(t *testing.T, status *v1beta1.KnativeServingStatus, wantReason, wantMsgContains string) {
	t.Helper()
	tc := status.GetCondition(base.TargetClusterResolved)
	if tc == nil || tc.Status != corev1.ConditionFalse {
		t.Fatalf("TargetClusterResolved: want False, got %v", tc)
	}
	if tc.Reason != wantReason {
		t.Fatalf("TargetClusterResolved.Reason: want %q, got %q", wantReason, tc.Reason)
	}
	if wantMsgContains != "" && !strings.Contains(tc.Message, wantMsgContains) {
		t.Fatalf("TargetClusterResolved.Message: want substring %q, got %q", wantMsgContains, tc.Message)
	}
	is := status.GetCondition(base.InstallSucceeded)
	if is != nil && is.Status == corev1.ConditionFalse {
		t.Fatalf("InstallSucceeded must not be flipped to False by MarkTargetClusterNotResolved: %v", is)
	}
}

func TestResolveTargetCluster_ReasonPropagation(t *testing.T) {
	newInstance := func() *v1beta1.KnativeServing {
		inst := &v1beta1.KnativeServing{
			ObjectMeta: metav1.ObjectMeta{Namespace: "knative-serving", Name: "default"},
			Spec: v1beta1.KnativeServingSpec{
				CommonSpec: base.CommonSpec{
					ClusterProfileRef: &base.ClusterProfileReference{
						Namespace: "fleet",
						Name:      "worker",
					},
				},
			},
		}
		inst.Status.InitializeConditions()
		return inst
	}
	runResolve := func(t *testing.T, provider *ClusterProvider, inst *v1beta1.KnativeServing) {
		t.Helper()
		manifest, err := mf.ManifestFrom(mf.Slice{})
		if err != nil {
			t.Fatalf("Failed to create manifest: %v", err)
		}
		var state ReconcileState
		stage := ResolveTargetCluster(provider, &state)
		if err := stage(context.Background(), &manifest, inst); err == nil {
			t.Fatal("ResolveTargetCluster() = nil, want error")
		}
	}

	t.Run("NilProvider", func(t *testing.T) {
		inst := newInstance()
		runResolve(t, nil, inst)
		assertTargetClusterNotResolved(t, &inst.Status,
			base.ReasonClusterProviderNotConfigured, "cluster provider not configured")
	})

	t.Run("Disabled", func(t *testing.T) {
		provider := &ClusterProvider{
			entries:       map[string]*clusterEntry{},
			access:        NoOpClusterProfileAccess{},
			ciClient:      fakeciclient.NewSimpleClientset(readyClusterProfile("fleet", "worker")),
			controllerCtx: context.Background(),
			remoteTimeout: defaultRemoteClusterTimeout,
		}
		inst := newInstance()
		runResolve(t, provider, inst)
		assertTargetClusterNotResolved(t, &inst.Status,
			base.ReasonMulticlusterDisabled, "multi-cluster support is disabled")
	})

	t.Run("ProfileNotFound", func(t *testing.T) {
		provider := newTestProviderWithStubAccess(&stubAccess{})
		inst := newInstance()
		runResolve(t, provider, inst)
		assertTargetClusterNotResolved(t, &inst.Status,
			base.ReasonClusterProfileNotFound, "failed to get ClusterProfile")
	})

	t.Run("ProfileNotReady", func(t *testing.T) {
		notReady := &clusterinventoryv1alpha1.ClusterProfile{
			ObjectMeta: metav1.ObjectMeta{Namespace: "fleet", Name: "worker"},
			Status: clusterinventoryv1alpha1.ClusterProfileStatus{
				Conditions: []metav1.Condition{
					{
						Type:               clusterinventoryv1alpha1.ClusterConditionControlPlaneHealthy,
						Status:             metav1.ConditionFalse,
						LastTransitionTime: metav1.Now(),
						Reason:             "Unhealthy",
						Message:            "control plane unhealthy",
					},
				},
			},
		}
		provider := newTestProviderWithStubAccess(&stubAccess{}, notReady)
		inst := newInstance()
		runResolve(t, provider, inst)
		assertTargetClusterNotResolved(t, &inst.Status,
			base.ReasonClusterProfileNotReady, "is not ready")
	})

	t.Run("AccessFailed", func(t *testing.T) {
		stub := &stubAccess{
			buildFn: func(*clusterinventoryv1alpha1.ClusterProfile) (*rest.Config, error) {
				return nil, errors.New("simulated exec-plugin failure")
			},
		}
		provider := newTestProviderWithStubAccess(stub, readyClusterProfile("fleet", "worker"))
		inst := newInstance()
		runResolve(t, provider, inst)
		assertTargetClusterNotResolved(t, &inst.Status,
			base.ReasonAccessProviderFailed, "failed to build config from ClusterProfile")
	})
}
