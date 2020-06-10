# Test

This directory contains tests and testing docs for `Serving Operator`:

- [Integration tests](#running-integration-tests) currently reside in
  [`/test/e2e`](./e2e)

## Running integration tests

Before running the integration, please make sure you have installed
`Serving Operator` by following the instruction [here](../README.md), and do not
install custom resource for operator or knative-serving installed in your
cluster.

Ensure required namespaces exist:

```bash
kubectl create namespace knative-serving
kubectl create namespace knative-eventing
```

To run all integration tests:

```bash
go test -v -tags=e2e -count=1 ./test/e2e
```
