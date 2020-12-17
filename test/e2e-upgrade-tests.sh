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
# Operator built from source. However, this script will download the latest
# source code of knative serving and generated the latest manifest file for
# serving installation. The serving operator will use this newly generated
# manifest file to replace the one under the directory
# cmd/manager/kodata/knative-serving. This purpose of this script is to verify
# whether the latest source code of operator can work properly with the latest
# source code of knative serving.

# If you already have a Knative cluster setup and kubectl pointing
# to it, call this script with the --run-tests arguments and it will use
# the cluster and run the tests.

# Calling this script without arguments will create a new cluster in
# project $PROJECT_ID, start knative in it, run the tests and delete the
# cluster.

export GO111MODULE=auto

source $(dirname $0)/e2e-common.sh

# TODO: remove when components can coexist in same namespace
export TEST_EVENTING_NAMESPACE=knative-eventing

function install_previous_operator_release() {
  install_istio || fail_test "Istio installation failed"
  install_operator
  install_previous_knative
}

function install_previous_knative() {
  header "Create the custom resources for Knative of the previous version"
  create_knative_serving ${PREVIOUS_SERVING_RELEASE_VERSION}
  create_knative_eventing ${PREVIOUS_EVENTING_RELEASE_VERSION}
}

function create_knative_serving() {
  version=${1}
  echo ">> Creating the custom resource of Knative Serving:"
  cat <<EOF | kubectl apply -f -
apiVersion: operator.knative.dev/v1alpha1
kind: KnativeServing
metadata:
  name: knative-serving
  namespace: ${TEST_NAMESPACE}
spec:
  version: "${version}"
EOF
}

function create_knative_eventing() {
  version=${1}
  echo ">> Creating the custom resource of Knative Eventing:"
  cat <<-EOF | kubectl apply -f -
apiVersion: operator.knative.dev/v1alpha1
kind: KnativeEventing
metadata:
  name: knative-eventing
  namespace: ${TEST_EVENTING_NAMESPACE}
spec:
  version: "${version}"
EOF
}

function create_latest_custom_resource() {
  echo ">> Creating the custom resource of Knative Serving:"
  cat <<EOF | kubectl apply -f -
apiVersion: operator.knative.dev/v1alpha1
kind: KnativeServing
metadata:
  name: knative-serving
  namespace: ${TEST_NAMESPACE}
EOF
  echo ">> Creating the custom resource of Knative Eventing:"
  cat <<-EOF | kubectl apply -f -
apiVersion: operator.knative.dev/v1alpha1
kind: KnativeEventing
metadata:
  name: knative-eventing
  namespace: ${TEST_EVENTING_NAMESPACE}
EOF
}

function knative_setup() {
  create_namespace
  install_previous_operator_release
}

# Create test resources and images
function test_setup() {
  test_setup_logging
  echo ">> Waiting for Ingress provider to be running..."
  if [[ -n "${ISTIO_VERSION}" ]]; then
    wait_until_pods_running istio-system || return 1
    wait_until_service_has_external_http_address istio-system istio-ingressgateway
  fi

  # Install kail if needed.
  if ! which kail >/dev/null; then
    bash <(curl -sfL https://raw.githubusercontent.com/boz/kail/master/godownloader.sh) -b "$GOPATH/bin"
  fi

  # Capture all logs.
  kail >${ARTIFACTS}/k8s.log.txt &
  local kail_pid=$!
  # Clean up kail so it doesn't interfere with job shutting down
  add_trap "kill $kail_pid || true" EXIT
}

# Skip installing istio as an add-on
initialize $@ --skip-istio-addon

TIMEOUT=10m
PROBE_TIMEOUT=20m
failed=0

header "Running preupgrade tests for Knative Operator"
go_test_e2e -tags=preupgrade -timeout=${TIMEOUT} ./test/upgrade || fail_test=1

header "Listing all the pods of the previous release"
wait_until_pods_running ${TEST_NAMESPACE}
wait_until_pods_running ${TEST_EVENTING_NAMESPACE}

create_latest_custom_resource

header "Running tests for Knative Operator"

# Run the postupgrade tests under operator
# Operator tests here will make sure that all the Knative deployments reach the desired states and operator CR is
# in ready state.
go_test_e2e -tags=postupgrade -timeout=${TIMEOUT} ./test/upgrade \
  --preservingversion="${PREVIOUS_SERVING_RELEASE_VERSION}" --preeventingversion="${PREVIOUS_EVENTING_RELEASE_VERSION}" || failed=1

header "Listing all the pods of the current release"
wait_until_pods_running ${TEST_NAMESPACE}
wait_until_pods_running ${TEST_EVENTING_NAMESPACE}

install_previous_knative

header "Running postdowngrade tests for Knative Operator"
go_test_e2e -tags=postdowngrade -timeout=${TIMEOUT} ./test/downgrade \
  --preservingversion="${PREVIOUS_SERVING_RELEASE_VERSION}" --preeventingversion="${PREVIOUS_EVENTING_RELEASE_VERSION}" || failed=1

header "Listing all the pods of the previous release"
wait_until_pods_running ${TEST_NAMESPACE}
wait_until_pods_running ${TEST_EVENTING_NAMESPACE}

# Require that tests succeeded.
(( failed )) && fail_test

success
