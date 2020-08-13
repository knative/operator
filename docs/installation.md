# Installation

- [Operator](#knative-operator)
- [Knative Serving](#knative-serving)
- [Knative Eventing](#knative-eventing)
- [Upgrades](#upgrades)

## Knative Operator

Before any Knative component can be installed, you must first install the
Knative Operator.

To install the latest release:

```
kubectl apply -f https://github.com/knative/operator/releases/latest/download/operator.yaml
```

Alternatively, the latest nightly build:

```
kubectl apply -f https://storage.googleapis.com/knative-nightly/operator/latest/operator.yaml
```

Once running, the operator will continuously watch for the following custom
resources:

- [KnativeServing](../config/300-serving.yaml) represents the
  [serving](https://knative.dev/development/serving/) component
- [KnativeEventing](../config/300-eventing.yaml) represents the
  [eventing](https://knative.dev/development/eventing/) component

Each custom resource includes an optional `spec` field you can set to
[customize your installation](configuration.md). Each also includes a `status`
field the operator updates with the progress of the component installation.

Creating the custom resource in a given namespace results in the installation of
the corresponding component's resources in the same namespace.

## Knative Serving

Unfortunately, the serving component currently requires Istio. If you don't have
it in your cluster,
[follow these instructions](https://knative.dev/development/install/installing-istio/)
before continuing.

The Knative Serving resources will be installed in whichever namespace you
create the `KnativeServing` instance. For the sake of simplicity, we'll use the
`default` namespace:

```sh
cat <<-EOF | kubectl apply -f -
apiVersion: operator.knative.dev/v1alpha1
kind: KnativeServing
metadata:
  name: ks
EOF
```

Once created, you should then see the Knative Serving pods coming up, and the
operator will update the `KnativeServing` instance's `status` field with the
progress of the installation:

```
kubectl get knativeserving ks -oyaml
```

To uninstall Knative Serving, simply delete the `KnativeServing` instance. This
will then trigger the operator to terminate all the serving pods and remove all
the serving resources.

```
kubectl delete knativeserving ks
```

## Knative Eventing

The eventing component is installed very similarly to serving:

```sh
cat <<-EOF | kubectl apply -f -
apiVersion: v1
kind: Namespace
metadata:
 name: knative-eventing
---
apiVersion: operator.knative.dev/v1alpha1
kind: KnativeEventing
metadata:
  name: ke
  namespace: knative-eventing
EOF
```

The operator will install the knative eventing resources in the same namespace
as the `KnativeEventing` instance and you can monitor its `status` field to see
its progress:

```
kubectl get knativeeventing -n knative-eventing ke -oyaml
```

And removing Knative Eventing is as simple as deleting the `KnativeEventing`
instance.

```
kubectl delete knativeeventing -n knative-eventing ke
```

# Upgrades

Upgrading the Knative operator will automatically trigger the upgrade of any
existing `KnativeServing` and `KnativeEventing` instances, so you may want to
create backups of them first:

```
kubectl get knativeserving --all-namespaces -oyaml >knativeserving.yaml
kubectl get knativeeventing --all-namespaces -oyaml >knativeeventing.yaml
```

Once you've created those backups, simply apply the new version of the operator
and your knative upgrade will begin immediately.

If something goes wrong, you should re-apply the previous version of the
operator, and then re-apply the backup files.
