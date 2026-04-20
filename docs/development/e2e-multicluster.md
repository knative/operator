# Running Multi-Cluster E2E Tests Locally

Run the multi-cluster (hub + spoke) E2E suite locally using
[kind](https://kind.sigs.k8s.io/). The automated wrapper is
`test/e2e-tests-multicluster.sh`; the steps below are the manual equivalents.

## Prerequisites

- `kind` (>= v0.31.0), `kubectl`, `ko`, `docker`, `envsubst` (from `gettext`).
- A working `$KO_DOCKER_REPO` (the script defaults to `kind.local`).
- Go toolchain matching `go.mod`.
- Host architecture: arm64 and amd64 are both supported.

All commands below are relative to the repository root.

## 1. Start hub and spoke kind clusters

```bash
export KIND_CLUSTER_NAME=kind
export SPOKE_CLUSTER_NAME=spoke
export SPOKE_KUBECONFIG=/tmp/spoke.kubeconfig
export SPOKE_HOST_KUBECONFIG=/tmp/spoke-host.kubeconfig

kind create cluster --name "${KIND_CLUSTER_NAME}" --wait 120s
kind create cluster --name "${SPOKE_CLUSTER_NAME}" \
  --kubeconfig "${SPOKE_HOST_KUBECONFIG}" --wait 120s

# Internal kubeconfig reachable from the hub operator pod via the docker bridge.
kind get kubeconfig --internal --name "${SPOKE_CLUSTER_NAME}" > "${SPOKE_KUBECONFIG}"
kind get kubeconfig          --name "${SPOKE_CLUSTER_NAME}" > "${SPOKE_HOST_KUBECONFIG}"
```

Both clusters should come up healthy:

```bash
kubectl                             get nodes
KUBECONFIG="${SPOKE_HOST_KUBECONFIG}" kubectl get nodes
```

## 2. Install the ClusterProfile CRD on the hub

```bash
: "${CLUSTER_INVENTORY_CRD_URL:=https://raw.githubusercontent.com/kubernetes-sigs/cluster-inventory-api/v0.1.0/config/crd/bases/multicluster.x-k8s.io_clusterprofiles.yaml}"
kubectl apply -f "${CLUSTER_INVENTORY_CRD_URL}"
kubectl wait --for=condition=Established --timeout=60s \
  crd/clusterprofiles.multicluster.x-k8s.io
```

## 3. Deploy the operator on the hub and wire up access provider config

Apply the operator from source:

```bash
ko apply -Rf config/
```

Generate a spoke bootstrap token and mount the access provider plumbing. The
helper script does this end to end:

```bash
source test/e2e-common.sh
install_access_provider_config   # builds and installs the token-exec-plugin
apply_cluster_profile default    # creates the ClusterProfile on the hub
```

A minimal provider config (written by the helper to
`/etc/cluster-inventory/config.json` inside the operator pod) looks like:

```json
{
  "providers": [
    {
      "name": "e2e-static-token",
      "execConfig": {
        "apiVersion": "client.authentication.k8s.io/v1",
        "command": "/etc/cluster-inventory/plugin/ko-app/token-exec-plugin",
        "args": ["/etc/cluster-inventory/access/token"],
        "interactiveMode": "Never"
      }
    }
  ]
}
```

Point the operator at the config via the CLI flag. The helper patches the
operator deployment; if you prefer to manage it yourself:

```bash
kubectl -n knative-operator patch deployment knative-operator \
  --type json \
  -p '[{"op":"add","path":"/spec/template/spec/containers/0/args/-","value":"--clusterprofile-provider-file=/etc/cluster-inventory/config.json"}]'
```

(Optional) Tune the remote deployments poll interval for a large fleet
simulation:

```bash
kubectl -n knative-operator patch deployment knative-operator \
  --type json \
  -p '[{"op":"add","path":"/spec/template/spec/containers/0/args/-","value":"--remote-deployments-poll-interval=30s"}]'
```

## 4. Apply a hub CR targeting the spoke

Create the manifest in a temp file (the repository does not ship a
`hack/manual/` directory) and apply it:

```bash
kubectl create ns knative-serving
cat > /tmp/knativeserving-spoke.yaml <<'EOF'
apiVersion: operator.knative.dev/v1beta1
kind: KnativeServing
metadata:
  name: knative-serving
  namespace: knative-serving
spec:
  clusterProfileRef:
    name: spoke
    namespace: default
EOF
kubectl apply -f /tmp/knativeserving-spoke.yaml
```

Same pattern for Eventing:

```bash
kubectl create ns knative-eventing
cat > /tmp/knativeeventing-spoke.yaml <<'EOF'
apiVersion: operator.knative.dev/v1beta1
kind: KnativeEventing
metadata:
  name: knative-eventing
  namespace: knative-eventing
spec:
  clusterProfileRef:
    name: spoke
    namespace: default
EOF
kubectl apply -f /tmp/knativeeventing-spoke.yaml
```

Watch both status conditions and the spoke anchor:

```bash
kubectl get knativeserving -A -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.status.conditions[?(@.type=="TargetClusterResolved")].status}{"\t"}{.status.conditions[?(@.type=="InstallSucceeded")].status}{"\n"}{end}'

KUBECONFIG="${SPOKE_HOST_KUBECONFIG}" kubectl get cm -A \
  -l operator.knative.dev/cr-name=knative-serving

KUBECONFIG="${SPOKE_HOST_KUBECONFIG}" kubectl -n knative-serving rollout status deploy/activator
```

## 5. Tear down and verify

Delete the hub CRs in reverse order; the operator's finalizer cleans the spoke:

```bash
kubectl delete knativeeventing -n knative-eventing knative-eventing
kubectl delete knativeserving  -n knative-serving  knative-serving

# Anchors should be gone.
KUBECONFIG="${SPOKE_HOST_KUBECONFIG}" kubectl get cm -A \
  -l operator.knative.dev/cr-name 2>/dev/null
```

## 6. Running the Go E2E suite

The test package is gated by two build tags:

```bash
go test -v -tags 'e2e multicluster' -count=1 \
  -run '^TestMulticluster' ./test/e2e
```

> Both tags are required: `e2e` enables the shared e2e bootstrap and
> `multicluster` enables only the spoke tests. Use a single space-separated
> string as shown above; `go test` does not accept comma-separated tag
> values or multiple `-tags` flags.

Environment variables read by the suite:

| Variable | Purpose | Default |
|----------|---------|---------|
| `SPOKE_CLUSTER_NAME` | `ClusterProfile.metadata.name` used by the tests | `spoke` |
| `SPOKE_CLUSTER_NAMESPACE` | `ClusterProfile.metadata.namespace` | `default` |
| `KUBECONFIG` | Hub kubeconfig (standard Go client discovery) | current context |
| `SPOKE_HOST_KUBECONFIG` | Spoke kubeconfig from the host's perspective | (required) |

If you want the end-to-end bootstrap in a single step, use the wrapper script,
which reuses the same helpers this guide calls manually:

```bash
./test/e2e-tests-multicluster.sh
```

## 7. Debugging tips

- Hub operator logs: `kubectl -n knative-operator logs deploy/knative-operator`.
  Look for `Remote deployments poll interval:` after the first reconcile of a
  `KnativeServing`/`KnativeEventing` CR to confirm the flag value, and
  `cluster provider closed during shutdown` on pod restarts (see
  `ClusterProviderClosed` in [multicluster.md](../multicluster.md#troubleshooting)).
- Spoke state dump: `KUBECONFIG="${SPOKE_HOST_KUBECONFIG}" kubectl get events -A --sort-by=.lastTimestamp`.
- If the `TargetClusterResolved` condition stays `False` with reason
  `AccessProviderFailed`, exec into the operator and run the plugin manually:
  ```bash
  kubectl -n knative-operator exec deploy/knative-operator -- \
    /etc/cluster-inventory/plugin/ko-app/token-exec-plugin \
    /etc/cluster-inventory/access/token
  ```

## 8. Cleanup

```bash
kind delete cluster --name "${SPOKE_CLUSTER_NAME}"
kind delete cluster --name "${KIND_CLUSTER_NAME}"
rm -f "${SPOKE_KUBECONFIG}" "${SPOKE_HOST_KUBECONFIG}"
```

## Why this matters before `knative/infra#827`

Until [knative/infra#827](https://github.com/knative/infra/issues/827) lands
the Prow job, local kind is the only CI-like environment for exercising
multi-cluster changes.
