# Knative Operator Release

Knative Operator officially releases the manifests at the [Github release page](https://github.com/knative/operator/releases) and the [Operatorhub.io](https://operatorhub.io/operator/knative-operator).
Please take the following steps to release Knative Operator:

- [How to release at the Github release page](#release-at-the-github)
- [How to release at the Operatorhub.io](#release-at-the-operatorhubio)

## Release at the Github

As the lead of the Knative Operation Work group or any other Knative Work Group, create a branch called release-{major}.{minor},
e.g. release-1.3, in the repository of [Knative Operator](https://github.com/knative/operator).

Once this new branch is created, Knative Prows will automatically launch the release process and publish the manifests
at the [release page](https://github.com/knative/operator/releases).

## Release at the Operatorhub.io

### Prerequisites

- Install Kustomize via the command:
```aidl
curl -s "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"  | bash
cp kustomize /usr/local/bin/kustomize
```

- [Install Operator-SDK via the command](https://sdk.operatorframework.io/docs/installation/)

### Generate the manifests to be submitted for Operatorhub.io

Set the version variable in the script "hack/generate-bundle.sh" to the version number to be release in the format
of major.minor.patch. For example, set the version to 1.3.0 with the following line:
```aidl
VERSION=1.3.0
```

Open a terminal, go to the home directory of the Knative Operator repository. Run the script:
```shell
./hack/generate-bundle.sh
```

Once this command is successfully run, a directory called bundle will be generated under the home directory of Knative
Operator repository locally. All the manifests under the directory bundle/manifests are all the artifacts to be submitted
to the operatorhub.io.

### Create a PR for k8s-operatorhub/community-operators

The process to publish a version on [OperatorHub](https://operatorhub.io/operator/knative-operator) is as below:
- Download the source code of `k8s-operatorhub/community-operators`.
- Locate the directory of Knative Operator at `community-operators/operators/knative-operator`.
- Create a directory named the new version, like 1.3.0, under `community-operators/operators/knative-operator`.
- Copy all the manifests under `knative-operator/bundle/manifests` to `community-operators/operators/knative-operator`.
- Modify the `file knative-operator.package.yaml` under `community-operators/operators/knative-operator`, bu changing
  the field `currentCSV` into `knative-operator.v{major}.{minor}.{patch}`, e.g. knative-operator.v1.3.0.
- Create a PR will all the changes above an submit to the repository `k8s-operatorhub/community-operators`.

Once the PR is reviewed and merged, Knative Operator of the new version is published in [Operatorhub.io](https://operatorhub.io/operator/knative-operator).

## Release the Helm chart

Open the file `hack/generate-helm.sh`, set the variable `VERSION` to the version of the Knative Operator, e.g. `VERSION=1.10.1`.

Open a terminal, go to the home directory of the Knative Operator repository. Run the script:
```shell
./hack/generate-helm.sh
```

This script will generate the templates at `charts/knative-operator/templates/operator.yaml`. There are two deployment
resources in this template: `knative-operator` and `operator-webhook`. Make sure you add the following environment variables
into the containers of these two deployments as below:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: knative-operator
  ...
spec:
  template:
    spec:
      containers:
        - name: knative-operator
        ...
          env:
            ...
            - name: KUBERNETES_MIN_VERSION
              value: "{{ .Values.knative_operator.kubernetes_min_version }}"
            ...      
```

and

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: operator-webhook
  ...
spec:
  template:
    spec:
      containers:
        - name: operator-webhook
        ...
          env:
            ...
            - name: KUBERNETES_MIN_VERSION
              value: "{{ .Values.knative_operator.kubernetes_min_version }}"
            ...      
```

You need to remove the line containing `logging.request-log-template:`, because the value of this key contains `{{ }}` in the example,
which leads to the error of `executing "knative-operator/templates/operator.yaml" at <.Request.Method>: nil pointer evaluating interface {}.Method`
for the command `helm install` to install the charts.

To package the charts into a .tgz artifact. Go to the directory `charts` in a terminal, and run the `helm package` command:
```shell
cd charts
helm package knative-operator
```

You will generate a file named `knative-operator-{version}.tgz`, under the directory `charts`. `{version}` is the version
of Knative Operator.

You can install the charts by running the command:
```shell
helm install knative-operator ./knative-operator-{version}.tgz
```

Replace `{version}` with the correct version for your artifact.
