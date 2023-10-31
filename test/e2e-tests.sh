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
# Operator built from source.  It is started by prow for each PR. For
# convenience, it can also be executed manually.

# If you already have a Knative cluster setup and kubectl pointing
# to it, call this script with the --run-tests arguments and it will use
# the cluster and run the tests.

# Calling this script without arguments will create a new cluster in
# project $PROJECT_ID, start knative in it, run the tests and delete the
# cluster.

source $(dirname $0)/e2e-common.sh

function knative_setup() {
  create_namespace
  install_operator
}

# Skip installing istio as an add-on
initialize $@ --skip-istio-addon

failed=0

# Run tests serially in the mesh and https scenarios.
E2E_TEST_FLAGS="${TEST_OPTIONS}"

function use_resolvable_domain() {
  # Temporarily turning off sslip.io tests, as DNS errors aren't always retried.
  echo "false"
}

if [ -z "${E2E_TEST_FLAGS}" ]; then
  E2E_TEST_FLAGS="-resolvabledomain=$(use_resolvable_domain) -ingress-class=${INGRESS_CLASS}"

  # Drop testing alpha and beta features with the Gateway API
  if [[ "${INGRESS_CLASS}" != *"gateway-api"* ]]; then
    E2E_TEST_FLAGS+=" -enable-alpha -enable-beta"
  fi
fi

cd ${KNATIVE_DIR}/"serving"
kubectl apply -f test/config/cluster-resources.yaml
go_test_e2e -timeout=30m \
  ./test/e2e \
  ${E2E_TEST_FLAGS} || failed=1

success
