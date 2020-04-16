# Knative Operator

Knative Operator is a project aiming to manage all
[Knative](https://knative.dev/) installation and upgrade in a consistent way
from a single codebase.

## Origins

This is a merger of the Knative
[serving-operator](https://github.com/knative/serving-operator) and the
[eventing-operator](https://github.com/knative/eventing-operator).

## Prerequisites

### Istio

On OpenShift, Istio will get installed automatically if not already present by
using the [Maistra Operator](https://maistra.io/).

For other platforms, see
[the docs](https://knative.dev/docs/install/installing-istio/)

### Operator SDK

This operator was originally created using the
[operator-sdk](https://github.com/operator-framework/operator-sdk/). It's not
strictly required but does provide some handy tooling.

## Installation

The following steps will install
[Knative Serving](https://github.com/knative/serving) and configure it
appropriately for your cluster in the `knative-serving` namespace. Please make
sure the [prerequisites](#Prerequisites) are installed first.

1. Install the operator

- Installing from source code:

To install from source code, run the command:

```
ko apply -f config/
```

- Installing a released version:

To install a released version of the operator go and download the latest
`operator.yaml` file from
[here](https://github.com/knative-sandbox/operator/releases) and apply it
(`kubectl apply -f operator.yaml`), or directly run:

```
kubectl apply -f https://github.com/knative-sandbox/operator/releases/download/v0.14.0/operator.yaml
```

2. Install the
   [KnativeServing custom resource](#the-knativeserving-custom-resource)

```sh
cat <<-EOF | kubectl apply -f -
apiVersion: v1
kind: Namespace
metadata:
 name: knative-serving
---
apiVersion: operator.knative.dev/v1alpha1
kind: KnativeServing
metadata:
  name: knative-serving
  namespace: knative-serving
spec:
  config:
    defaults:
      revision-timeout-seconds: "300"  # 5 minutes
    autoscaler:
      stable-window: "60s"
    deployment:
      registriesSkippingTagResolving: "ko.local,dev.local"
    logging:
      loglevel.controller: "debug"
EOF
```

3. Install the
   [KnativeEventing custom resource](#the-knativeeventing-custom-resource)

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
  name: knative-eventing
  namespace: knative-eventing
EOF
```

## The `KnativeServing` Custom Resource

The installation of Knative Serving is triggered by the creation of a
`KnativeServing` custom resource (CR) as defined by
[this CRD](config/300-operator-v1alpha1-knative-crd.yaml). The operator
will deploy Knative Serving in the same namespace containing the
`KnativeServing` CR, and this CR will trigger the installation, reconfiguration,
or removal of the knative serving resources.

The optional `spec.config` field can be used to set the corresponding entries in
the Knative Serving ConfigMaps. Conditions for a successful install and
available deployments will be updated in the `status` field, as well as which
version of Knative Serving the operator installed.

The following are all equivalent:

```
kubectl get knativeservings.operator.knative.dev -oyaml
kubectl get knativeserving -oyaml
```

To uninstall Knative Serving, simply delete the `KnativeServing` resource.

```
kubectl delete knativeserving --all
```

## The `KnativeEventing` Custom Resource

The installation of Knative Eventing is triggered by the creation of a
`KnativeEventing` custom resource (CR) as defined by
[this CRD](config/300-operator-v1alpha1-knative-crd.yaml). The operator
will deploy Knative Eventing in the same namespace containing the
`KnativeEventing` CR, and this CR will trigger the installation,
reconfiguration, or removal of the knative eventing resources.

The following are all equivalent:

```
kubectl get knativeeventings.operator.knative.dev
kubectl get knativeeventing
```

To uninstall Knative Eventing, simply delete the `KnativeEventing` resource.

```
kubectl delete knativeeventings --all
```

## Configure Knative with Operator Custom Resources

- [Configure Knative Serving using Operator](docs/install/operator/configuring-serving-cr.md)
- [Configure Knative Eventing using Operator](docs/install/operator/configuring-eventing-cr.md)

## Upgrade

Please refer to the [upgrade guide](docs/upgrade_guide.md) for a safe upgrade
process.

