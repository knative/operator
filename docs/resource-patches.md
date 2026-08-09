# Customizing generated resources with patches

Cluster administrators can use `spec.patches` on a `KnativeServing` or
`KnativeEventing` resource when an existing typed override does not cover the
generated Kubernetes resource they need to change. Patches run after the
Operator's built-in transformations and apply in declaration order. Each patch
sees the result of earlier patches, so a later patch can override their changes.

## Patch target

Each patch must identify exactly one generated resource by API version, kind,
and name. Set `namespace` only when resources in more than one namespace have
the same identity. Reconciliation fails if the target matches zero or multiple
resources; this makes a renamed or removed target visible during an upgrade.

```yaml
spec:
  patches:
  - target:
      apiVersion: autoscaling/v2
      kind: HorizontalPodAutoscaler
      name: webhook
    patch:
      type: strategic
      content: |
        spec:
          minReplicas: 2
          maxReplicas: 10
```

Patches apply to all resources assembled for the component, including Serving
or Eventing core resources, enabled Ingress or Source resources, extension
resources, and `additionalManifests`.

| Patch type | Use it for | Content format |
| --- | --- | --- |
| `strategic` | Kubernetes resources with named lists, such as Deployment containers | Strategic Merge Patch in YAML or JSON |
| `merge` | Maps and scalar fields, including custom resources | RFC 7386 Merge Patch in YAML or JSON |
| `json` | Exact add, replace, or remove operations | RFC 6902 JSON Patch in YAML or JSON |

Strategic Merge Patch requires a Kubernetes type registered in the Operator's
scheme. Use `merge` or `json` for an unregistered custom resource. A patch may
not change the target's API version, kind, name, or namespace.

The `$patch` key is reserved for `type: strategic`. The Operator rejects it at
any depth in a `merge` or `json` patch instead of applying it as literal
resource data. JSON Patch users can replace a field with the RFC 6902
`replace` operation:

```yaml
patch:
  type: json
  content: |
    - op: replace
      path: /spec/replicas
      value: 4
```

## Letting KEDA manage replicas

Knative 1.23 includes the following HorizontalPodAutoscalers. An ingress or
broker HPA is present only when its corresponding optional component is
enabled.

| Custom resource | Workload | HPA name |
| --- | --- | --- |
| `KnativeServing` | `activator` | `activator` |
| `KnativeServing` | `webhook` | `webhook` |
| `KnativeServing` with Kourier | `3scale-kourier-gateway` | `3scale-kourier-gateway` |
| `KnativeEventing` | `eventing-webhook` | `eventing-webhook` |
| `KnativeEventing` with MTChannelBasedBroker | `mt-broker-ingress` | `broker-ingress-hpa` |
| `KnativeEventing` with MTChannelBasedBroker | `mt-broker-filter` | `broker-filter-hpa` |

Run `kubectl get hpa -n knative-serving` or `kubectl get hpa -n knative-eventing`
to confirm which HPAs are present in the cluster.

To replace the bundled `activator` HPA with KEDA, remove the HPA and leave
`spec.replicas` unmanaged by the Operator:

```yaml
apiVersion: operator.knative.dev/v1beta1
kind: KnativeServing
metadata:
  name: knative-serving
  namespace: knative-serving
spec:
  patches:
  - target:
      apiVersion: autoscaling/v2
      kind: HorizontalPodAutoscaler
      name: activator
    patch:
      type: strategic
      content: |
        $patch: delete
  - target:
      apiVersion: apps/v1
      kind: Deployment
      name: activator
    patch:
      type: strategic
      content: |
        spec:
          replicas: null
```

Resource removal is available only with `type: strategic`. The directive keeps
the exact HPA identity absent. Configure the external autoscaler to use another
HPA name. Removing the delete patch causes the bundled HPA to be created again
on the next reconciliation.

A root-level `$patch: delete` cannot target a CustomResourceDefinition or
Namespace. Knative Serving also requires its `apps/v1` Deployment named
`webhook` during installation, so that Deployment cannot be deleted.
