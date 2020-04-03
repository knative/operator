# Upgrade Guide

This document describes how to upgrade the serving-operator to an expected
version.

## Backup precaution

As an administrator, you are recommended to save the content of the custom
resource for serving-operator before upgrading your operator. Make sure that you
know the name and the namespace of your CR, and use the following command to
save the CR in a file called `serving_operator_cr.yaml`:

For the version v0.10.0 or later:

```
kubectl get KnativeServing <cr-name> -n <namespace> -o=yaml > serving_operator_cr.yaml
```

Replace `<cr-name>` with the name of your CR, and `<namespace>` with the
namespace.

One version of serving-operator installs only one specific version of Knative
Serving. With your operator successfully upgraded, your Knative Serving is
upgraded as well.

## v0.10.0, v0.11.x -> v0.12.0

Both of v0.10 and v0.11 versions are able to upgrade to the version v0.12.0 by
running the following command:

```
kubectl apply -f https://github.com/knative/serving-operator/releases/download/v0.12.0/serving-operator.yaml
```
