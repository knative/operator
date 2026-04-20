# Multi-Cluster Deployment

The operator can deploy Knative Serving and Eventing to remote clusters from a
single hub cluster. A `KnativeServing` or `KnativeEventing` CR carrying a
`spec.clusterProfileRef` reconciles on the referenced spoke cluster; without it
the operator behaves as before.

The hub needs network access to each spoke API server. Connection details are
resolved through the Cluster Inventory API (`ClusterProfile`).

> Note: if direct connectivity is not available, reverse the direction with
> [OCM Cluster Proxy](https://open-cluster-management.io/docs/getting-started/integration/cluster-proxy/).

## Prerequisites

- **Kubernetes 1.35+** on the hub cluster (image volumes must be available).
- The **Cluster Inventory API** CRD (`ClusterProfile`) installed on the hub.
- Network connectivity from the operator pod to each spoke API server.

## Usage

Set `spec.clusterProfileRef` on a CR to target a remote cluster:

```yaml
apiVersion: operator.knative.dev/v1beta1
kind: KnativeServing
metadata:
  name: knative-serving
  namespace: knative-serving
spec:
  clusterProfileRef:
    name: spoke-cluster-1
    namespace: fleet-system
```

The operator resolves the `ClusterProfile`, builds a `rest.Config` via the
configured access provider, and applies manifests on the spoke. A
`TargetClusterResolved` status condition tracks whether the remote cluster was
reached.

`--clusterprofile-provider-file` must point to an access provider config JSON
file (`sigs.k8s.io/cluster-inventory-api/pkg/access`); without it, any CR with
a `clusterProfileRef` will fail to reconcile.

## Helm chart

Enable multi-cluster in `values.yaml`:

```yaml
knative_operator:
  multicluster:
    enabled: true
    accessProvidersConfig:
      providers:
        - name: token-secretreader
          execConfig:
            apiVersion: client.authentication.k8s.io/v1
            command: /access-plugins/token-secretreader/kubeconfig-secretreader-plugin
            provideClusterInfo: true
    plugins:
      - name: token-secretreader
        image: ghcr.io/example/plugin:v1.0.0
        mountPath: /access-plugins/token-secretreader
```

The chart creates a `ConfigMap` with the provider config and mounts each
plugin as a Kubernetes image volume inside the operator pod.

## Namespace configuration

`spec.namespaceConfiguration.labels` and `spec.namespaceConfiguration.annotations`
are applied to the spoke namespace when the operator creates it. Existing
spoke namespaces are not modified.

## Anchor ConfigMap

For remote deployments, the operator creates an anchor ConfigMap
(`{kind}-{cr-name}-root-owner`) on the spoke. Namespace-scoped resources use
it as their `OwnerReference`, so deleting the anchor triggers GC of all owned
resources. Cluster-scoped resources are not owned by the anchor and are
cleaned up by `FinalizeRemoteCluster` when the hub CR is deleted.

The anchor carries an `operator.knative.dev/protected=true` annotation and a
description annotation warning against manual deletion. To uninstall safely,
delete the corresponding CR on the hub.

## Remote deployments poll interval

While spoke deployments roll out, the operator requeues the CR to re-check
readiness. The interval is controlled by `--remote-deployments-poll-interval`
(default `10s`); values below `1s` fall back to the default. The effective
value is logged at operator startup.

Larger values reduce reconcile traffic on hubs managing many spokes, at the
cost of slower observability of readiness transitions:

| Spoke count | Recommended interval |
|-------------|----------------------|
| < 10 | `10s` (default) |
| 10-100 | `30s` |
| > 100 | `60s` |

### Setting the interval

```yaml
knative_operator:
  multicluster:
    enabled: true
    remoteDeploymentsPollInterval: 30s
```

## Troubleshooting

Check the status condition on the CR:

```bash
kubectl get knativeserving -n <ns> <name> -o jsonpath='{.status.conditions[?(@.type=="TargetClusterResolved")]}'
```

Common reasons for `TargetClusterResolved=False`:

- **ClusterProfileNotFound**: the referenced `ClusterProfile` does not exist.
  Check `spec.clusterProfileRef`.
- **ClusterProfileNotReady**: `ClusterProfile` exists but is unhealthy.
  Inspect `kubectl get clusterprofile -n <ns> <name> -o yaml`.
- **ClusterProfileUnavailable**: fetch failed or cache not primed yet.
  Transient during bootstrap; check hub API server reachability if persistent.
- **AccessProviderFailed**: exec plugin error or timeout. Check operator logs.
- **AccessProviderNotConfigured**: `--clusterprofile-provider-file` is not set
  on the operator Deployment.
- **MulticlusterDisabled**: the provider config declares no provider for the
  `ClusterProfile`. Add a matching provider entry.
- **RemoteClientCreationFailed**: valid `rest.Config` received but client
  construction failed (typically invalid TLS material or unreachable host).
- **RemoteClusterStale**: cached spoke connection was invalidated (context
  cancelled, TCP drop). The next reconcile refreshes and recovers.
- **ClusterProviderClosed**: operator is shutting down; the next leader
  re-reconciles and recovers.

If spoke deployments are not coming up, confirm `TargetClusterResolved=True`,
check the operator logs on the hub, and inspect the spoke cluster directly with
`kubectl --kubeconfig=<spoke> get deployments -n <ns>`.
