# Knative Operator Development

Most of the same tools required for [Knative Serving
development](https://github.com/knative/serving/blob/master/DEVELOPMENT.md)
are required for the operator, too.

You should clone this repo to `$GOPATH/src/knative.dev/operator`. All
commands below are relative to that path.

To install the operator:

```
ko apply -f config/
```

To run the unit tests:

```
go test -v ./...
```

To run the e2e tests:

```
ko apply -f config/
kubectl create namespace knative-serving
go test -v -tags=e2e -count=1 ./test/e2e
```
