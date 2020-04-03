#!/usr/bin/env bash

# Copyright 2019 The Knative Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This script runs the end-to-end tests against Knative Serving
# Operator built from source.  It is started by Travis CI for each
# PR. For convenience, it can also be executed manually.

source $(dirname $0)/e2e-common.sh

knative_setup

# Let's see what the operator did
kubectl logs deployment/knative-serving-operator
kubectl get pod --all-namespaces
kubectl get knativeserving --all-namespaces -o yaml

test_setup

NODE_PORT=$(kubectl get svc istio-ingressgateway -n istio-system -o jsonpath="{.spec.ports[?(@.port==80)].nodePort}")
NODE_IP=$(kubectl get node -o jsonpath="{.items[0].status.addresses[?(@.type=='InternalIP')].address}")

echo ">> Detected ingress gateway $NODE_IP:$NODE_PORT"

# Run the tests
header "Running tests"

failed=0

echo ">> Testing basic helloworld-go"
# Run a basic test to ensure we can deploy a Knative service
cat <<EOF | kubectl apply -f - || failed=1
apiVersion: serving.knative.dev/v1alpha1
kind: Service
metadata:
  name: helloworld-go
  namespace: $TEST_NAMESPACE
spec:
  template:
    spec:
      containers:
      - image: gcr.io/knative-samples/helloworld-go
        resources:
          requests:
            cpu: 25m
EOF

wait_until_routable "$NODE_IP:$NODE_PORT" "helloworld-go.$TEST_NAMESPACE.example.com" || failed=1
curl -f -H "Host: helloworld-go.$TEST_NAMESPACE.example.com" http://$NODE_IP:$NODE_PORT/ || failed=1
(( !failed )) && echo ">> PASS: basic helloworld-go"

# Require that all tests succeeded.
(( failed )) && fail_test

test_teardown

success
