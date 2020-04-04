# Knative Eventing Operator

[![Go Report Card](https://goreportcard.com/badge/knative/eventing-operator)](https://goreportcard.com/report/knative/eventing-operator)
[![Releases](https://img.shields.io/github/release-pre/knative/eventing-operator.svg)](https://github.com/knative/eventing-operator/releases)
[![LICENSE](https://img.shields.io/github/license/knative/eventing-operator.svg)](https://github.com/knative/eventing-operator/blob/master/LICENSE)
[![Slack Status](https://img.shields.io/badge/slack-join_chat-white.svg?logo=slack&style=social)](https://knative.slack.com)

Knative Eventing Operator is a project aiming to deploy and manage Knative
Eventing in an automated way.

The following steps will install
[Knative Eventing](https://github.com/knative/Eventing) and configure it
appropriately for your cluster in the `knative-eventing` namespace. Please make
sure the [prerequisites](#Prerequisites) are installed first.

1. Install the operator

   To install from source code, run the command:

   ```
   ko apply -f config/
   ```

   To install from an existing image, change the value of `image` into
   `quay.io/openshift-knative/knative-eventing-operator:v0.6.0` or any other
   valid operator image in the file config/operator.yaml, and run the following
   command:

   ```
   kubectl apply -f config/
   ```

1. Install the [Eventing custom resource](#the-eventing-custom-resource)

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

Please refer to [Building the Operator Image](#building-the-operator-image) to
build your own image.

## Prerequisites

- [`ko`](https://github.com/google/ko)

  Install `ko` with the following command, if it is not available on your
  machine:

  ```
  go get -u github.com/google/ko/cmd/ko
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

Pass `--help` for further details on the various subcommands

## Building the Operator Image

To build the operator with `ko`, configure your an environment variable
`KO_DOCKER_REPO` as the docker repository to which developer images should be
pushed (e.g. `gcr.io/[gcloud-project]`, `docker.io/[username]`,
`quay.io/[repo-name]`, etc).

Then, build the operator image:

```
ko publish knative.dev/eventing-operator/cmd/manager -t $VERSION
```

You need to access the image by the name
`KO_DOCKER_REPO/manager-[md5]:$VERSION`, which you are able to find in the
output of the above `ko publish` command.

The image should match what's in [config/operator.yaml](config/operator.yaml)
and the `$VERSION` should match [version.go](version/version.go) and correspond
to the contents of [config/](config/).

## Upgrade

Please refer to the [upgrade guide](doc/upgrade_guide.md) for a safe upgrade
process.

## Operator Lifecycle Manager and OperatorHub

Knative Eventing operator has the metadata in Operator Lifecycle Manager (OLM)
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
  name: knative-eventing-operator-sub
  generateName: knative-eventing-operator-
  namespace: $OPERATOR_NS
spec:
  source: knative-eventing-operator
  sourceNamespace: $OLM_NS
  name: knative-eventing-operator
  channel: alpha
EOF
```
