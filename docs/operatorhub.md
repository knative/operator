# Publishing a new version on OperatorHub

## Overview

The process to publish a version on [OperatorHub](https://operatorhub.io/) is as
following:

- Prepare the files in `knative/operator` repository
- Test locally
- Send a PR to https://github.com/operator-framework/
- Once that passes the Prow tests and gets approved by the OperatorHub people,
  send a PR to `knative/operator` so that we also have the metadata handy in
  this repository

## Instructions for metadata files:

- Copy and paste the directory for the previous version in
  `deploy/olm-catalog/knative-operator/` into a new directory
- Replace `knativeservings.operator.knative.dev.crd.yaml` with the YAML of
  `KnativeServing` CRD in the new version
- Replace `knativeeventings.operator.knative.dev.crd.yaml` with the YAML of
  `KnativeEventing` CRD in the new version
- `*.clusterserviceversion.yaml` file's name should be changed for the new
  version
- Make sure following is correct in `*.clusterserviceversion.yaml`:
  - `metadata.annotations.createdAt`: should point to the date of the release
  - `metadata.annotations.containerImage`: should point to the new container
    image of the operator
  - `metadata.name`: should be the operator name and version
  - `spec.install.spec.deployments`: should reflect the deployment structure in
    `config/operator.yaml`
  - `spec.install.spec.permissions`: should reflect the RBAC structure in
    `config/role.yaml`
  - `spec.replaces`: should be the version that this version is replacing
  - `spec.version`: should be the operator version
- Change `currentCSV` in
  `deploy/olm-catalog/knative-operator/knative-operator.package.yaml` to the new
  version

## Testing locally

You need to install OLM on your cluster. There are instructions to install OLM
from master available at
<https://github.com/operator-framework/operator-lifecycle-manager/blob/master/doc/install/install.md>.

For installing stable releases, visit the
[releases page](https://github.com/operator-framework/operator-lifecycle-manager/releases)
where you can find instructions as well.

Also run OLM console as it provides a nice and easy way to install operators
with OLM. Instructions available
[here](https://github.com/operator-framework/operator-lifecycle-manager#user-interface).

Make sure you don't have any `catalogsource` and `operatorsource` CRs on your cluster
after installation. There are some default ones created for operators available on
OperatorHub, but these might get mixed with the operator versions we are going to
install from source.  

### Testing a fresh installation

- Install the operator catalog source by running
  `./hack/generate-olm-catalog-source.sh | kubectl apply -n olm -f -`.
- Install the operator via OLM console by clicking _OperatorHub -> Knative
  Operator -> Install_
- Create `Knative Serving` and/or `Knative Eventing` CRDs
- You should see Knative Serving and Eventing installed successfully.

### Testing an upgrade

- Change the `currentCSV` field in `deploy/olm-catalog-knative-operator/knative-operator.package.yaml` file to the previous version for the current channel
- Create another channel in that file with a `currentCSV` value of the current version
- Do the steps above as you are installing a fresh installation with the old version. Make sure you select the channel for the previous version when installing the operator.
- Change the operator subscription channel on the console using _Installed Operators -> Knative Operator -> Subscription -> Channel_
- Approve the `InstallPlan`
- The operator will upgrade to the latest version and it should reconcile the KnativeServing and KnativeEventing CRDs

## PR to OperatorHub

- Clone https://github.com/operator-framework/
- Copy and paste the new directory under `deploy/olm-catalog-knative-operator/` into `upstream-community-operators/knative-operator/` in `operator-framework` repository
- Also copy paste `deploy/olm-catalog-knative-operator/knative-operator.package.yaml` into `upstream-community-operators/knative-operator/knative-operator.package.yaml` in `operator-framework` repository

## PR to this repo

Wait until the PR in OperatorHub is merged as OperatorSDK team will validate the created files.

## Outcome

The new operator version will be available at
https://operatorhub.io/operator/knative-operator
