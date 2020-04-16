# Knative Combined Operator


Knative Operator is a project aiming to manage all
[Knative](https://knative.dev/) installation and upgrade in a consistent way
from a single codebase.

## Origins

This is a merger of the Knative
[serving-operator](https://github.com/knative/serving-operator) and the
[eventing-operator](https://github.com/knative/eventing-operator).

## Status

This project is currently in sandbox (incubating) state.

Contact the WG leads:

- Vincent Hou

For details.

# Knative Serving Operator

[![Go Report Card](https://goreportcard.com/badge/knative/serving-operator)](https://goreportcard.com/report/knative/serving-operator)
[![Releases](https://img.shields.io/github/release-pre/knative/serving-operator.svg)](https://github.com/knative/serving-operator/releases)
[![LICENSE](https://img.shields.io/github/license/knative/serving-operator.svg)](https://github.com/knative/serving-operator/blob/master/LICENSE)
[![Slack Status](https://img.shields.io/badge/slack-join_chat-white.svg?logo=slack&style=social)](https://knative.slack.com)

Knative Serving Operator is a project aiming to deploy and manage Knative
Serving in an automated way.

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
`serving-operator.yaml` file from
[here](https://github.com/knative/serving-operator/releases) and apply it
(`kubectl apply -f serving-operator.yaml`), or directly run:

```
kubectl apply -f https://github.com/knative/serving-operator/releases/download/v0.10.0/serving-operator.yaml
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

Please refer to [Building the Operator Image](#building-the-operator-image) to
build your own image.

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

## Development

It can be convenient to run the operator outside of the cluster to test changes.
The following command will build the operator and use your current "kube config"
to connect to the cluster:

```
./hack/run-local.sh
```

Pass `--help` for further details on the various subcommands

## Building the Operator Image

To build the operator with `ko`, configure your an environment variable
`KO_DOCKER_REPO` as the docker repository to which developer images should be
pushed (e.g. `gcr.io/[gcloud-project]`, `docker.io/[username]`,
`quay.io/[repo-name]`, etc).

Install `ko` with the following command, if it is not available on your machine:

```
go get -u github.com/google/ko/cmd/ko
```

Then, build the operator image:

```
ko publish knative.dev/serving-operator/cmd/manager -t $VERSION
```

You need to access the image by the name
`KO_DOCKER_REPO/manager-[md5]:$VERSION`, which you are able to find in the
output of the above `ko publish` command.

The image should match what's in [config/operator.yaml](config/operator.yaml)
and the `$VERSION` should match [version.go](version/version.go) and correspond
to the contents of [config/](config/).

## Operator Lifecycle Manager and OperatorHub

Knative Serving operator has the metadata in Operator Lifecycle Manager (OLM)
checked in at `deploy/olm-catalog`. Files in there are for reference purposes
and also for testing and tooling.

In order to install the operator `CatalogSource` to a
[cluster with OLM](https://github.com/operator-framework/operator-lifecycle-manager/blob/master/doc/install/install.md),
run these commands:

```
OLM_NS=$(kubectl get deploy --all-namespaces | grep olm-operator | awk '{print $1}')
./hack/generate-olm-catalog-source.sh | kubectl apply -n $OLM_NS -f -
```

Then install the operator by creating a subscription:

```
OLM_NS=$(kubectl get operatorgroups --all-namespaces | grep olm-operators | awk '{print $1}')
OPERATOR_NS=$(kubectl get operatorgroups --all-namespaces | grep global-operators | awk '{print $1}')

cat <<-EOF | kubectl apply -f -
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: knative-serving-operator-sub
  generateName: knative-serving-operator-
  namespace: $OPERATOR_NS
spec:
  source: knative-serving-operator
  sourceNamespace: $OLM_NS
  name: knative-serving-operator
  channel: alpha
EOF
```

## Configure Knative with Operator Custom Resources

- [Configure Knative Serving using Operator](.docs/install/operator/configuring-serving-cr.md)
- [Configure Knative Eventing using Operator](.docs/install/operator/configuring-eventing-cr.md)

## Upgrade

Please refer to the [upgrade guide](doc/upgrade_guide.md) for a safe upgrade
process.

