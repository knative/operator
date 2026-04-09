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
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	mfc "github.com/manifestival/client-go-client"
	mf "github.com/manifestival/manifestival"
	"golang.org/x/sync/singleflight"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"knative.dev/operator/pkg/apis/operator/base"
	"knative.dev/pkg/logging"

	clusterinventoryv1alpha1 "sigs.k8s.io/cluster-inventory-api/apis/v1alpha1"
	clusterinventoryclient "sigs.k8s.io/cluster-inventory-api/client/clientset/versioned"
	"sigs.k8s.io/cluster-inventory-api/pkg/access"
)

const (
	defaultRemoteClusterTimeout = 10 * time.Second
	remoteClusterQPS            = float32(20)
	remoteClusterBurst          = 40
)

var (
	errClusterNotResolved = errors.New("ClusterProfile not yet resolved")
	errClusterStale       = errors.New("cluster connection is stale")

	errMulticlusterDisabled = errors.New(
		"multi-cluster support is disabled: set --clusterprofile-provider-file to enable")

	globalProviderMu sync.Mutex
	globalProvider   *ClusterProvider
)

// ClusterProfileAccess builds a rest.Config from a ClusterProfile.
type ClusterProfileAccess interface {
	BuildConfigFromCP(cp *clusterinventoryv1alpha1.ClusterProfile) (*rest.Config, error)
}

type NoOpClusterProfileAccess struct{}

func (NoOpClusterProfileAccess) BuildConfigFromCP(*clusterinventoryv1alpha1.ClusterProfile) (*rest.Config, error) {
	return nil, errMulticlusterDisabled
}

var _ ClusterProfileAccess = (*access.Config)(nil)

// RemoteClusterClients provides clients for a resolved remote cluster.
type RemoteClusterClients interface {
	MfClient() mf.Client
	KubeClient() kubernetes.Interface
	RestConfig() *rest.Config
}

type clusterEntry struct {
	mfClient   mf.Client
	kubeClient kubernetes.Interface
	restConfig *rest.Config
	cancel     context.CancelFunc
	ctx        context.Context
	closeOnce  sync.Once
}

func (e *clusterEntry) MfClient() mf.Client              { return e.mfClient }
func (e *clusterEntry) KubeClient() kubernetes.Interface { return e.kubeClient }
func (e *clusterEntry) RestConfig() *rest.Config         { return e.restConfig }
func (e *clusterEntry) IsAlive() bool                    { return e.ctx.Err() == nil }

func (e *clusterEntry) Close() {
	e.closeOnce.Do(func() {
		e.cancel()
		if e.kubeClient == nil {
			return
		}
		if rc := e.kubeClient.Discovery().RESTClient(); rc != nil {
			if restClient, ok := rc.(*rest.RESTClient); ok && restClient.Client != nil {
				restClient.Client.CloseIdleConnections()
			}
		}
	})
}

// ClusterProvider resolves ClusterProfile references into cached client sets.
type ClusterProvider struct {
	mu            sync.RWMutex
	entries       map[string]*clusterEntry
	access        ClusterProfileAccess
	ciClient      clusterinventoryclient.Interface
	controllerCtx context.Context
	group         singleflight.Group
	remoteTimeout time.Duration

	listenersMu     sync.RWMutex
	listeners       []ClusterProfileListener
	informerStarted bool
}

func NewClusterProvider(
	controllerCtx context.Context,
	localConfig *rest.Config,
	providerFile string,
) (*ClusterProvider, error) {
	ciClient, err := clusterinventoryclient.NewForConfig(localConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create cluster-inventory client: %w", err)
	}

	var accessImpl ClusterProfileAccess
	if providerFile == "" {
		accessImpl = NoOpClusterProfileAccess{}
		logging.FromContext(controllerCtx).Info(
			"multi-cluster support disabled (--clusterprofile-provider-file is empty)")
	} else {
		cfg, err := access.NewFromFile(providerFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load access provider config from %q: %w", providerFile, err)
		}
		accessImpl = cfg
	}

	p := &ClusterProvider{
		entries:       make(map[string]*clusterEntry),
		access:        accessImpl,
		ciClient:      ciClient,
		controllerCtx: controllerCtx,
		remoteTimeout: defaultRemoteClusterTimeout,
	}

	go func() {
		<-controllerCtx.Done()
		p.CloseAll()
	}()

	return p, nil
}

func GetOrCreateClusterProvider(
	controllerCtx context.Context,
	localConfig *rest.Config,
	providerFile string,
) (*ClusterProvider, error) {
	globalProviderMu.Lock()
	defer globalProviderMu.Unlock()
	if globalProvider != nil {
		return globalProvider, nil
	}
	p, err := NewClusterProvider(controllerCtx, localConfig, providerFile)
	if err != nil {
		return nil, err
	}
	globalProvider = p
	return p, nil
}

func (c *ClusterProvider) Refresh(ctx context.Context, namespace, name string) (string, error) {
	key := namespace + "/" + name
	callerLogger := logging.FromContext(ctx)
	v, err, _ := c.group.Do(key, func() (any, error) {
		refreshCtx, cancel := context.WithTimeout(c.controllerCtx, c.remoteTimeout)
		defer cancel()
		refreshCtx = logging.WithLogger(refreshCtx, callerLogger)
		reason, err := c.doRefresh(refreshCtx, key, namespace, name)
		return reason, err
	})
	reason, _ := v.(string)
	return reason, err
}

func (c *ClusterProvider) doRefresh(ctx context.Context, key, namespace, name string) (string, error) {
	logger := logging.FromContext(ctx)

	cp, err := c.ciClient.ApisV1alpha1().ClusterProfiles(namespace).Get(
		ctx, name, metav1.GetOptions{})
	if err != nil {
		c.Remove(key)
		if apierrors.IsNotFound(err) {
			return "ClusterProfileNotFound",
				fmt.Errorf("failed to get ClusterProfile %s: %w", key, err)
		}
		return "ClusterProfileUnavailable",
			fmt.Errorf("failed to get ClusterProfile %s: %w", key, err)
	}

	if !isClusterProfileReady(cp) {
		c.Remove(key)
		logger.Infof("ClusterProfile %s is not ready", key)
		return "ClusterProfileNotReady",
			fmt.Errorf("ClusterProfile %s is not ready", key)
	}

	newConfig, err := c.access.BuildConfigFromCP(cp)
	if err != nil {
		c.Remove(key)
		if errors.Is(err, errMulticlusterDisabled) {
			return "MulticlusterDisabled",
				fmt.Errorf("failed to build config from ClusterProfile %s: %w", key, err)
		}
		return "AccessProviderFailed",
			fmt.Errorf("failed to build config from ClusterProfile %s: %w", key, err)
	}
	newConfig.Timeout = c.remoteTimeout
	newConfig.QPS = remoteClusterQPS
	newConfig.Burst = remoteClusterBurst

	c.mu.RLock()
	if existing, ok := c.entries[key]; ok && configEqual(existing.restConfig, newConfig) {
		c.mu.RUnlock()
		return "", nil
	}
	c.mu.RUnlock()

	mfClient, err := mfc.NewClient(newConfig)
	if err != nil {
		return "RemoteClientCreationFailed",
			fmt.Errorf("failed to create remote manifestival client: %w", err)
	}
	kubeClient, err := kubernetes.NewForConfig(newConfig)
	if err != nil {
		return "RemoteClientCreationFailed",
			fmt.Errorf("failed to create remote kube client: %w", err)
	}

	clusterCtx, cancel := context.WithCancel(c.controllerCtx) //nolint:gosec // cancel is stored in clusterEntry and called via Close()

	newEntry := &clusterEntry{
		mfClient:   mfClient,
		kubeClient: kubeClient,
		restConfig: newConfig,
		cancel:     cancel,
		ctx:        clusterCtx,
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if existing, ok := c.entries[key]; ok {
		existing.Close()
		logger.Infof("ClusterProfile %s access changed, refreshing cached clients", key)
	} else {
		logger.Infof("ClusterProfile %s resolved, caching clients for %s", key, newConfig.Host)
	}
	c.entries[key] = newEntry
	return "", nil
}

func (c *ClusterProvider) Remove(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if entry, ok := c.entries[key]; ok {
		entry.Close()
		delete(c.entries, key)
	}
}

func (c *ClusterProvider) CloseAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for key, entry := range c.entries {
		entry.Close()
		delete(c.entries, key)
	}
}

// ClusterProfileListener receives notifications when a ClusterProfile changes.
type ClusterProfileListener struct {
	ListCRs    func(namespace, name string) []types.NamespacedName
	EnqueueKey func(types.NamespacedName)
}

func (c *ClusterProvider) RegisterListener(l ClusterProfileListener) {
	c.listenersMu.Lock()
	defer c.listenersMu.Unlock()
	c.listeners = append(c.listeners, l)
}

func (c *ClusterProvider) notifyListeners(namespace, name string) {
	c.listenersMu.RLock()
	defer c.listenersMu.RUnlock()
	for _, l := range c.listeners {
		for _, key := range l.ListCRs(namespace, name) {
			l.EnqueueKey(key)
		}
	}
}

func (c *ClusterProvider) Get(ctx context.Context, clusterName string) (RemoteClusterClients, string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[clusterName]
	if !ok {
		return nil, "ClusterProfileUnavailable",
			fmt.Errorf("%w: %s", errClusterNotResolved, clusterName)
	}
	if !entry.IsAlive() {
		return nil, "RemoteClusterStale",
			fmt.Errorf("%w: %s", errClusterStale, clusterName)
	}
	return entry, "", nil
}

func (c *ClusterProvider) GetOrRefresh(ctx context.Context, namespace, name string) (RemoteClusterClients, string, error) {
	key := namespace + "/" + name

	if entry, _, err := c.Get(ctx, key); err == nil {
		return entry, "", nil
	}

	if reason, err := c.Refresh(ctx, namespace, name); err != nil {
		return nil, reason, err
	}
	return c.Get(ctx, key)
}

func configEqual(a, b *rest.Config) bool {
	return a.Host == b.Host &&
		reflect.DeepEqual(a.TLSClientConfig, b.TLSClientConfig) &&
		reflect.DeepEqual(a.ExecProvider, b.ExecProvider) &&
		a.BearerToken == b.BearerToken &&
		a.BearerTokenFile == b.BearerTokenFile &&
		a.Username == b.Username &&
		a.Password == b.Password &&
		reflect.DeepEqual(a.AuthProvider, b.AuthProvider) &&
		reflect.DeepEqual(a.Impersonate, b.Impersonate)
}

func isClusterProfileReady(cp *clusterinventoryv1alpha1.ClusterProfile) bool {
	cond := apimeta.FindStatusCondition(cp.Status.Conditions,
		clusterinventoryv1alpha1.ClusterConditionControlPlaneHealthy)
	if cond == nil {
		return false
	}
	return cond.Status == metav1.ConditionTrue
}

// ShouldFinalizeClusterScoped returns true when no other KComponent targets the same cluster.
func ShouldFinalizeClusterScoped(
	components []base.KComponent,
	original base.KComponent,
) bool {
	for _, comp := range components {
		if comp.GetDeletionTimestamp().IsZero() &&
			SameClusterProfile(comp.GetSpec().GetClusterProfileRef(), original.GetSpec().GetClusterProfileRef()) {
			return false
		}
	}
	return true
}

func SameClusterProfile(a, b *base.ClusterProfileReference) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.Namespace == b.Namespace && a.Name == b.Name
}

func FinalizeRemoteCluster(
	ctx context.Context,
	clients RemoteClusterClients,
	manifest *mf.Manifest,
	instance base.KComponent,
	optionalPreds ...mf.Predicate,
) error {
	var errs []error

	if err := DeleteAnchorConfigMap(ctx, clients.KubeClient(), instance); err != nil {
		errs = append(errs, fmt.Errorf("anchor ConfigMap deletion: %w", err))
	}

	if manifest != nil {
		manifest.Client = clients.MfClient()
		clusterScoped := mf.Predicate(func(u *unstructured.Unstructured) bool {
			return u.GetNamespace() == ""
		})

		if len(optionalPreds) > 0 {
			optionalPred := mf.Any(optionalPreds...)

			if err := manifest.Filter(mf.NoCRDs, mf.Not(optionalPred), clusterScoped).
				Delete(mf.IgnoreNotFound(true)); err != nil {
				errs = append(errs, fmt.Errorf("cluster-scoped deletion: %w", err))
			}
			if err := manifest.Filter(mf.NoCRDs, optionalPred, clusterScoped).
				Delete(mf.IgnoreNotFound(true)); err != nil && !apimeta.IsNoMatchError(err) {
				errs = append(errs, fmt.Errorf("optional cluster-scoped deletion: %w", err))
			}
		} else {
			if err := manifest.Filter(mf.NoCRDs, clusterScoped).
				Delete(mf.IgnoreNotFound(true)); err != nil {
				errs = append(errs, fmt.Errorf("cluster-scoped deletion: %w", err))
			}
		}
	}

	return errors.Join(errs...)
}

// FinalizeRemoteClusterIfNeeded runs remote finalization if the instance targets a remote cluster.
func FinalizeRemoteClusterIfNeeded(
	ctx context.Context,
	provider *ClusterProvider,
	manifest *mf.Manifest,
	instance base.KComponent,
	optionalPreds ...mf.Predicate,
) (bool, error) {
	cpRef := instance.GetSpec().GetClusterProfileRef()
	if cpRef == nil {
		return false, nil
	}
	if provider == nil {
		return true, fmt.Errorf("cluster provider not configured but clusterProfileRef is set")
	}
	logger := logging.FromContext(ctx)
	clusterName := cpRef.Namespace + "/" + cpRef.Name
	entry, reason, err := provider.Get(ctx, clusterName)
	if err != nil {
		if errors.Is(err, errClusterNotResolved) {
			logger.Warnf("ClusterProfile %s not yet resolved; remote resources may be orphaned. "+
				"Remove finalizer manually if cluster is permanently gone.", clusterName)
		} else if errors.Is(err, errClusterStale) {
			logger.Warnf("ClusterProfile %s connection stale; remote resources may be orphaned. "+
				"Will retry on next reconcile.", clusterName)
		}
		instance.GetStatus().MarkTargetClusterNotResolved(
			reason,
			fmt.Sprintf("ClusterProfile %s unavailable: %v", clusterName, err))
		return true, err
	}
	if err := FinalizeRemoteCluster(ctx, entry, manifest, instance, optionalPreds...); err != nil {
		return true, fmt.Errorf("remote finalization: %w", err)
	}
	return true, nil
}

func AnchorName(instance base.KComponent) string {
	kind := strings.ToLower(instance.GroupVersionKind().Kind)
	return kind + "-" + instance.GetName() + "-root-owner"
}

const maxResourceNameLength = 253

// EnsureAnchorConfigMap creates or updates the anchor ConfigMap on the remote cluster.
func EnsureAnchorConfigMap(
	ctx context.Context,
	kubeClient kubernetes.Interface,
	instance base.KComponent,
) (*corev1.ConfigMap, error) {
	name := AnchorName(instance)
	if len(name) > maxResourceNameLength {
		return nil, fmt.Errorf("anchor ConfigMap name %q exceeds maximum length of %d characters; shorten the CR name", name, maxResourceNameLength)
	}
	ns := instance.GetNamespace()

	if _, err := kubeClient.CoreV1().Namespaces().Get(ctx, ns, metav1.GetOptions{}); err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("failed to check namespace %s: %w", ns, err)
		}
		nsObj := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
				Labels: map[string]string{
					"app.kubernetes.io/managed-by": "knative-operator",
				},
			},
		}
		if _, err := kubeClient.CoreV1().Namespaces().Create(ctx, nsObj, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
			return nil, fmt.Errorf("failed to create namespace %s on remote cluster: %w", ns, err)
		}
	}

	expectedLabels := map[string]string{
		"app.kubernetes.io/managed-by": "knative-operator",
		"operator.knative.dev/cr-name": instance.GetName(),
	}
	expectedAnnotations := map[string]string{
		"operator.knative.dev/anchor": "true",
	}

	anchor, err := kubeClient.CoreV1().ConfigMaps(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("failed to get anchor ConfigMap %s/%s: %w", ns, name, err)
		}
		anchor = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:        name,
				Namespace:   ns,
				Labels:      expectedLabels,
				Annotations: expectedAnnotations,
			},
		}
		anchor, err = kubeClient.CoreV1().ConfigMaps(ns).Create(ctx, anchor, metav1.CreateOptions{})
		if err == nil {
			return anchor, nil
		}
		if !apierrors.IsAlreadyExists(err) {
			return nil, fmt.Errorf("failed to create anchor ConfigMap %s/%s: %w", ns, name, err)
		}
		anchor, err = kubeClient.CoreV1().ConfigMaps(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to get existing anchor ConfigMap %s/%s: %w", ns, name, err)
		}
	}

	needsUpdate := false
	if anchor.Labels == nil {
		anchor.Labels = make(map[string]string)
	}
	for k, v := range expectedLabels {
		if anchor.Labels[k] != v {
			anchor.Labels[k] = v
			needsUpdate = true
		}
	}
	if anchor.Annotations == nil {
		anchor.Annotations = make(map[string]string)
	}
	for k, v := range expectedAnnotations {
		if anchor.Annotations[k] != v {
			anchor.Annotations[k] = v
			needsUpdate = true
		}
	}
	if needsUpdate {
		anchor, err = kubeClient.CoreV1().ConfigMaps(ns).Update(ctx, anchor, metav1.UpdateOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to update anchor ConfigMap %s/%s: %w", ns, name, err)
		}
	}

	return anchor, nil
}

func DeleteAnchorConfigMap(
	ctx context.Context,
	kubeClient kubernetes.Interface,
	instance base.KComponent,
) error {
	name := AnchorName(instance)
	ns := instance.GetNamespace()
	err := kubeClient.CoreV1().ConfigMaps(ns).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete anchor ConfigMap %s/%s: %w", ns, name, err)
	}
	return nil
}

func ResolveTargetCluster(provider *ClusterProvider, state *ReconcileState) Stage {
	return func(ctx context.Context, manifest *mf.Manifest, instance base.KComponent) error {
		cpRef := instance.GetSpec().GetClusterProfileRef()
		if cpRef == nil {
			instance.GetStatus().MarkTargetClusterResolved()
			return nil
		}

		if provider == nil {
			instance.GetStatus().MarkTargetClusterNotResolved(
				"ClusterProviderNotConfigured",
				"cluster provider not configured; set --clusterprofile-provider-file")
			return fmt.Errorf("cluster provider not configured but clusterProfileRef is set")
		}

		entry, reason, err := provider.GetOrRefresh(ctx, cpRef.Namespace, cpRef.Name)
		if err != nil {
			instance.GetStatus().MarkTargetClusterNotResolved(reason, err.Error())
			return fmt.Errorf("failed to resolve target cluster: %w", err)
		}
		instance.GetStatus().MarkTargetClusterResolved()

		manifest.Client = entry.MfClient()
		state.RemoteClients = entry

		anchor, err := EnsureAnchorConfigMap(ctx, entry.KubeClient(), instance)
		if err != nil {
			return fmt.Errorf("failed to ensure anchor ConfigMap: %w", err)
		}
		anchor.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("ConfigMap"))
		state.AnchorOwner = anchor
		return nil
	}
}
