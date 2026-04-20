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
	"sync/atomic"
	"testing"
	"time"

	mf "github.com/manifestival/manifestival"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"

	"knative.dev/operator/pkg/apis/operator/base"
	"knative.dev/operator/pkg/apis/operator/v1beta1"

	clusterinventoryv1alpha1 "sigs.k8s.io/cluster-inventory-api/apis/v1alpha1"
)

func TestResolveTargetCluster_NilRef(t *testing.T) {
	instance := &v1beta1.KnativeServing{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
	}
	instance.Status.InitializeConditions()

	manifest, err := mf.ManifestFrom(mf.Slice{})
	if err != nil {
		t.Fatalf("Failed to create manifest: %v", err)
	}

	origClient := manifest.Client

	var state ReconcileState
	stage := ResolveTargetCluster(nil, &state)
	if err := stage(context.Background(), &manifest, instance); err != nil {
		t.Fatalf("ResolveTargetCluster() = %v, want nil", err)
	}

	if manifest.Client != origClient {
		t.Fatal("manifest.Client changed unexpectedly")
	}

	if state.AnchorOwner != nil {
		t.Fatal("state.AnchorOwner is non-nil, want nil")
	}

	if state.IsRemote() {
		t.Fatal("state.IsRemote() = true, want false")
	}

	cond := instance.Status.GetCondition(base.TargetClusterResolved)
	if cond == nil || cond.Status != corev1.ConditionTrue {
		t.Fatalf("TargetClusterResolved = %v, want True", cond)
	}
}

func TestEnsureAnchorConfigMap_Create(t *testing.T) {
	kubeClient := fake.NewSimpleClientset()
	instance := &v1beta1.KnativeServing{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-ns",
			Name:      "test",
		},
	}

	ctx := context.Background()
	anchor, err := EnsureAnchorConfigMap(ctx, kubeClient, instance)
	if err != nil {
		t.Fatalf("EnsureAnchorConfigMap() error: %v", err)
	}

	expectedName := "knativeserving-test-root-owner"
	if anchor.Name != expectedName {
		t.Fatalf("anchor.Name = %q, want %q", anchor.Name, expectedName)
	}
	if anchor.Namespace != "test-ns" {
		t.Fatalf("anchor.Namespace = %q, want %q", anchor.Namespace, "test-ns")
	}

	if got := anchor.Labels["app.kubernetes.io/managed-by"]; got != "knative-operator" {
		t.Fatalf("label managed-by = %q, want %q", got, "knative-operator")
	}
	if got := anchor.Labels["operator.knative.dev/cr-name"]; got != "test" {
		t.Fatalf("label cr-name = %q, want %q", got, "test")
	}

	if got := anchor.Annotations["operator.knative.dev/anchor"]; got != "true" {
		t.Fatalf("annotation anchor = %q, want %q", got, "true")
	}
	if _, ok := anchor.Annotations["operator.knative.dev/warning"]; ok {
		t.Fatal("unexpected warning annotation on newly created anchor")
	}

	ns, err := kubeClient.CoreV1().Namespaces().Get(ctx, "test-ns", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Get namespace: %v", err)
	}
	if got := ns.Labels["app.kubernetes.io/managed-by"]; got != "knative-operator" {
		t.Fatalf("namespace label managed-by = %q, want %q", got, "knative-operator")
	}
}

func TestEnsureAnchorConfigMap_AlreadyExists(t *testing.T) {
	existingAnchor := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "knativeserving-test-root-owner",
			Namespace: "test-ns",
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "knative-operator",
				"operator.knative.dev/cr-name": "test",
			},
			Annotations: map[string]string{
				"operator.knative.dev/anchor": "true",
			},
		},
	}
	existingNS := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "test-ns"},
	}
	kubeClient := fake.NewSimpleClientset(existingNS, existingAnchor)

	instance := &v1beta1.KnativeServing{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-ns",
			Name:      "test",
		},
	}

	ctx := context.Background()
	anchor, err := EnsureAnchorConfigMap(ctx, kubeClient, instance)
	if err != nil {
		t.Fatalf("EnsureAnchorConfigMap() error: %v", err)
	}

	if anchor.Name != "knativeserving-test-root-owner" {
		t.Fatalf("anchor.Name = %q, want %q", anchor.Name, "knativeserving-test-root-owner")
	}
	if anchor.Namespace != "test-ns" {
		t.Fatalf("anchor.Namespace = %q, want %q", anchor.Namespace, "test-ns")
	}
}

func TestEnsureAnchorConfigMap_AdditiveMerge(t *testing.T) {
	instance := &v1beta1.KnativeServing{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "knative-serving",
			Namespace: "knative-serving",
		},
	}
	oldAnchor := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      AnchorName(instance),
			Namespace: "knative-serving",
			Labels: map[string]string{
				"old-label": "old-value",
			},
			Annotations: map[string]string{
				"old-annotation": "old-value",
			},
		},
	}
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "knative-serving"},
	}
	kubeClient := fake.NewSimpleClientset(oldAnchor, ns)

	anchor, err := EnsureAnchorConfigMap(context.Background(), kubeClient, instance)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := anchor.Labels["app.kubernetes.io/managed-by"]; got != "knative-operator" {
		t.Errorf("managed-by label = %q, want %q", got, "knative-operator")
	}
	if got := anchor.Labels["operator.knative.dev/cr-name"]; got != "knative-serving" {
		t.Errorf("cr-name label = %q, want %q", got, "knative-serving")
	}
	if got := anchor.Labels["old-label"]; got != "old-value" {
		t.Errorf("old-label not preserved, got labels: %v", anchor.Labels)
	}

	if got := anchor.Annotations["operator.knative.dev/anchor"]; got != "true" {
		t.Errorf("anchor annotation = %q, want %q", got, "true")
	}
	if got := anchor.Annotations["old-annotation"]; got != "old-value" {
		t.Errorf("old-annotation not preserved, got annotations: %v", anchor.Annotations)
	}
}

func TestEnsureAnchorConfigMap_NameTooLong(t *testing.T) {
	longName := strings.Repeat("a", 250)
	instance := &v1beta1.KnativeServing{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-ns",
			Name:      longName,
		},
	}

	kubeClient := fake.NewSimpleClientset()
	_, err := EnsureAnchorConfigMap(context.Background(), kubeClient, instance)
	if err == nil {
		t.Fatal("EnsureAnchorConfigMap() = nil, want error")
	}
	if !strings.Contains(err.Error(), "exceeds maximum length") {
		t.Fatalf("error = %v, want substring %q", err, "exceeds maximum length")
	}
}

func TestDeleteAnchorConfigMap_Success(t *testing.T) {
	existingAnchor := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "knativeserving-test-root-owner",
			Namespace: "test-ns",
		},
	}
	kubeClient := fake.NewSimpleClientset(existingAnchor)

	instance := &v1beta1.KnativeServing{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-ns",
			Name:      "test",
		},
	}

	ctx := context.Background()
	if err := DeleteAnchorConfigMap(ctx, kubeClient, instance); err != nil {
		t.Fatalf("DeleteAnchorConfigMap() error: %v", err)
	}

	_, err := kubeClient.CoreV1().ConfigMaps("test-ns").Get(ctx, "knativeserving-test-root-owner", metav1.GetOptions{})
	if err == nil {
		t.Fatal("anchor ConfigMap still exists after deletion")
	}
}

func TestConfigEqual(t *testing.T) {
	cfg := &rest.Config{
		Host:            "https://example.com",
		BearerToken:     "token",
		BearerTokenFile: "/path/to/token",
		Username:        "user",
		Password:        "pass",
	}
	same := &rest.Config{
		Host:            "https://example.com",
		BearerToken:     "token",
		BearerTokenFile: "/path/to/token",
		Username:        "user",
		Password:        "pass",
	}
	if !configEqual(cfg, same) {
		t.Fatal("configEqual() = false, want true")
	}

	if configEqual(cfg, &rest.Config{Host: "https://other.com", BearerToken: "token"}) {
		t.Fatal("configEqual(different Host) = true, want false")
	}

	if configEqual(cfg, &rest.Config{Host: "https://example.com", BearerToken: "other-token"}) {
		t.Fatal("configEqual(different BearerToken) = true, want false")
	}

	if configEqual(cfg, &rest.Config{
		Host:            "https://example.com",
		TLSClientConfig: rest.TLSClientConfig{CertFile: "/path/to/cert"},
	}) {
		t.Fatal("configEqual(different TLSClientConfig) = true, want false")
	}
}

func TestSameClusterProfile(t *testing.T) {
	tests := []struct {
		name string
		a, b *base.ClusterProfileReference
		want bool
	}{
		{
			name: "both nil",
			a:    nil,
			b:    nil,
			want: true,
		},
		{
			name: "a nil",
			a:    nil,
			b:    &base.ClusterProfileReference{Namespace: "ns", Name: "name"},
			want: false,
		},
		{
			name: "same",
			a:    &base.ClusterProfileReference{Namespace: "ns", Name: "name"},
			b:    &base.ClusterProfileReference{Namespace: "ns", Name: "name"},
			want: true,
		},
		{
			name: "different",
			a:    &base.ClusterProfileReference{Namespace: "ns", Name: "name1"},
			b:    &base.ClusterProfileReference{Namespace: "ns", Name: "name2"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SameClusterProfile(tt.a, tt.b)
			if got != tt.want {
				t.Fatalf("SameClusterProfile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShouldFinalizeClusterScoped(t *testing.T) {
	ref := &base.ClusterProfileReference{Namespace: "fleet", Name: "spoke1"}

	tests := []struct {
		name       string
		components []base.KComponent
		original   base.KComponent
		want       bool
	}{
		{
			name:       "no other components",
			components: []base.KComponent{},
			original: &v1beta1.KnativeServing{
				ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: "ks"},
			},
			want: true,
		},
		{
			name: "another alive component with same cluster profile",
			components: []base.KComponent{
				&v1beta1.KnativeServing{
					ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: "ks-other"},
					Spec: v1beta1.KnativeServingSpec{
						CommonSpec: base.CommonSpec{ClusterProfileRef: ref},
					},
				},
			},
			original: &v1beta1.KnativeServing{
				ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: "ks"},
				Spec: v1beta1.KnativeServingSpec{
					CommonSpec: base.CommonSpec{ClusterProfileRef: ref},
				},
			},
			want: false,
		},
		{
			name: "another alive component with different cluster profile",
			components: []base.KComponent{
				&v1beta1.KnativeServing{
					ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: "ks-other"},
					Spec: v1beta1.KnativeServingSpec{
						CommonSpec: base.CommonSpec{ClusterProfileRef: &base.ClusterProfileReference{
							Namespace: "fleet", Name: "spoke2",
						}},
					},
				},
			},
			original: &v1beta1.KnativeServing{
				ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: "ks"},
				Spec: v1beta1.KnativeServingSpec{
					CommonSpec: base.CommonSpec{ClusterProfileRef: ref},
				},
			},
			want: true,
		},
		{
			name: "both local (nil refs), another alive",
			components: []base.KComponent{
				&v1beta1.KnativeServing{
					ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: "ks-other"},
				},
			},
			original: &v1beta1.KnativeServing{
				ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: "ks"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldFinalizeClusterScoped(tt.components, tt.original)
			if got != tt.want {
				t.Fatalf("ShouldFinalizeClusterScoped() = %v, want %v", got, tt.want)
			}
		})
	}
}

// stubClientFactory returns predetermined clients without network I/O.
type stubClientFactory struct {
	mfErr      error
	kubeErr    error
	kubeClient kubernetes.Interface
	mfCount    atomic.Int32
	kubeCount  atomic.Int32
}

func (s *stubClientFactory) NewMfClient(*rest.Config) (mf.Client, error) {
	s.mfCount.Add(1)
	if s.mfErr != nil {
		return nil, s.mfErr
	}
	return fakeMfClient{}, nil
}

func (s *stubClientFactory) NewKubeClient(*rest.Config) (kubernetes.Interface, error) {
	s.kubeCount.Add(1)
	if s.kubeErr != nil {
		return nil, s.kubeErr
	}
	if s.kubeClient != nil {
		return s.kubeClient, nil
	}
	return fake.NewSimpleClientset(), nil
}

// fakeMfClient is a no-op manifestival client for tests that don't exercise mf I/O.
type fakeMfClient struct{}

func (fakeMfClient) Create(_ *unstructured.Unstructured, _ ...mf.ApplyOption) error { return nil }
func (fakeMfClient) Update(_ *unstructured.Unstructured, _ ...mf.ApplyOption) error { return nil }
func (fakeMfClient) Delete(_ *unstructured.Unstructured, _ ...mf.DeleteOption) error {
	return nil
}

func (fakeMfClient) Get(_ *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	return nil, nil
}

var _ mf.Client = fakeMfClient{}

// blockingAccess holds BuildConfigFromCP open until release() is closed; used for dedup testing.
type blockingAccess struct {
	entered chan struct{}
	release chan struct{}

	mu    sync.Mutex
	count int
	seen  bool
}

func (b *blockingAccess) BuildConfigFromCP(*clusterinventoryv1alpha1.ClusterProfile) (*rest.Config, error) {
	b.mu.Lock()
	b.count++
	first := !b.seen
	b.seen = true
	b.mu.Unlock()
	if first {
		close(b.entered)
	}
	<-b.release
	return &rest.Config{Host: "https://blocked.example.com"}, nil
}

func (b *blockingAccess) calls() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.count
}

func TestDoRefresh_Concurrency_SameKeyDeduped(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	access := &blockingAccess{
		entered: make(chan struct{}),
		release: make(chan struct{}),
	}
	factory := &stubClientFactory{}
	provider := newTestProviderWithStubAccess(&stubAccess{}, readyClusterProfile("fleet", "worker"))
	provider.access = access
	provider.clientFactory = factory

	var wg sync.WaitGroup
	wg.Add(2)
	errs := make([]error, 2)

	go func() {
		defer wg.Done()
		_, err := provider.Refresh(ctx, "fleet", "worker")
		errs[0] = err
	}()

	select {
	case <-access.entered:
	case <-time.After(5 * time.Second):
		close(access.release)
		t.Fatal("leader goroutine did not enter BuildConfigFromCP within 5s")
	}

	followerReady := make(chan struct{})
	go func() {
		defer wg.Done()
		close(followerReady)
		_, err := provider.Refresh(ctx, "fleet", "worker")
		errs[1] = err
	}()
	<-followerReady

	// A second in-flight call would still be blocked in BuildConfigFromCP; on dedup calls() stays at 1.
	deadlineCtx, cancelDeadline := context.WithTimeout(context.Background(), 500*time.Millisecond)
	t.Cleanup(cancelDeadline)
	for deadlineCtx.Err() == nil {
		if access.calls() != 1 {
			break
		}
		select {
		case <-time.After(10 * time.Millisecond):
		case <-deadlineCtx.Done():
		}
	}
	if got := access.calls(); got != 1 {
		close(access.release)
		wg.Wait()
		t.Fatalf("BuildConfigFromCP calls before release = %d, want 1", got)
	}

	close(access.release)
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("goroutine %d: Refresh() = %v, want nil", i, err)
		}
	}
	if got := access.calls(); got != 1 {
		t.Fatalf("BuildConfigFromCP final calls = %d, want 1", got)
	}
	if got := factory.mfCount.Load(); got != 1 {
		t.Fatalf("NewMfClient calls = %d, want 1", got)
	}
	if got := factory.kubeCount.Load(); got != 1 {
		t.Fatalf("NewKubeClient calls = %d, want 1", got)
	}
}

func TestDoRefresh_Concurrency_DifferentKeysIndependent(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	factory := &stubClientFactory{}
	provider := newTestProviderWithStubAccess(
		&stubAccess{},
		readyClusterProfile("fleet", "worker-a"),
		readyClusterProfile("fleet", "worker-b"),
	)
	provider.clientFactory = factory

	var wg sync.WaitGroup
	wg.Add(2)
	errs := make([]error, 2)
	names := []string{"worker-a", "worker-b"}
	for i, name := range names {
		go func() {
			defer wg.Done()
			_, err := provider.Refresh(ctx, "fleet", name)
			errs[i] = err
		}()
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("goroutine %d (%s): Refresh() = %v, want nil", i, names[i], err)
		}
	}
	if got := factory.mfCount.Load(); got != 2 {
		t.Fatalf("NewMfClient calls = %d, want 2", got)
	}
	if got := factory.kubeCount.Load(); got != 2 {
		t.Fatalf("NewKubeClient calls = %d, want 2", got)
	}
}

func TestDoRefresh_ClientCreationFailure_Manifestival(t *testing.T) {
	provider := newTestProviderWithStubAccess(&stubAccess{}, readyClusterProfile("fleet", "worker"))
	mfErr := errors.New("mf boom")
	factory := &stubClientFactory{mfErr: mfErr}
	provider.clientFactory = factory

	reason, err := provider.Refresh(context.Background(), "fleet", "worker")
	if err == nil {
		t.Fatal("Refresh() = nil, want error")
	}
	if reason != base.ReasonRemoteClientCreationFailed {
		t.Errorf("reason = %q, want %q", reason, base.ReasonRemoteClientCreationFailed)
	}
	if !errors.Is(err, mfErr) {
		t.Errorf("error chain does not wrap mfErr: %v", err)
	}
	if !strings.Contains(err.Error(), "manifestival") {
		t.Errorf("error message = %q, want it to mention %q", err.Error(), "manifestival")
	}
	if got := factory.mfCount.Load(); got != 1 {
		t.Errorf("NewMfClient calls = %d, want 1", got)
	}
	if got := factory.kubeCount.Load(); got != 0 {
		t.Errorf("NewKubeClient calls = %d, want 0 (should short-circuit before kube client)", got)
	}
	if got := len(provider.entries); got != 0 {
		t.Errorf("provider.entries size = %d, want 0 (failure must not cache)", got)
	}
}

func TestDoRefresh_ClientCreationFailure_Kube(t *testing.T) {
	provider := newTestProviderWithStubAccess(&stubAccess{}, readyClusterProfile("fleet", "worker"))
	kubeErr := errors.New("kube boom")
	factory := &stubClientFactory{kubeErr: kubeErr}
	provider.clientFactory = factory

	reason, err := provider.Refresh(context.Background(), "fleet", "worker")
	if err == nil {
		t.Fatal("Refresh() = nil, want error")
	}
	if reason != base.ReasonRemoteClientCreationFailed {
		t.Errorf("reason = %q, want %q", reason, base.ReasonRemoteClientCreationFailed)
	}
	if !errors.Is(err, kubeErr) {
		t.Errorf("error chain does not wrap kubeErr: %v", err)
	}
	if !strings.Contains(err.Error(), "kube") {
		t.Errorf("error message = %q, want it to mention %q", err.Error(), "kube")
	}
	if got := factory.mfCount.Load(); got != 1 {
		t.Errorf("NewMfClient calls = %d, want 1", got)
	}
	if got := factory.kubeCount.Load(); got != 1 {
		t.Errorf("NewKubeClient calls = %d, want 1", got)
	}
	if got := len(provider.entries); got != 0 {
		t.Errorf("provider.entries size = %d, want 0 (failure must not cache)", got)
	}
}
