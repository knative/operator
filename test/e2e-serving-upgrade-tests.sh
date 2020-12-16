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

function knative_setup() {
  create_namespace
  install_previous_operator_release
  download_knative "${KNATIVE_SERVING_REPO:-knative/serving}" serving "${KNATIVE_REPO_BRANCH}"
}

# Create test resources and images
function test_setup() {
  echo ">> Creating test resources (test/config/) in Knative Serving repository"
  cd ${KNATIVE_DIR}/serving
  for i in $(ls test/config/*.yaml); do
    sed s/knative-serving/${TEST_NAMESPACE}/ $i | ko apply ${KO_FLAGS} -f -
  done || return 1
  # Disable the chaosduck deployment as in Serving upgrade prow
  kubectl -n "${TEST_NAMESPACE}" scale deployment "chaosduck" --replicas=0 || fail_test

  echo ">> Uploading test images..."
  # We only need to build and publish two images among all the test images
  ${OPERATOR_DIR}/test/upload-test-images.sh ${KNATIVE_DIR}/serving "test/test_images/pizzaplanetv1"
  ${OPERATOR_DIR}/test/upload-test-images.sh ${KNATIVE_DIR}/serving "test/test_images/pizzaplanetv2"
  ${OPERATOR_DIR}/test/upload-test-images.sh ${KNATIVE_DIR}/serving "test/test_images/autoscale"

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

  cd ${OPERATOR_DIR}
}

# This function either generate the manifest based on a branch or download the latest manifest for Knative Serving.
# Parameter: $1 - branch name. If it is empty, download the manifest from nightly build.

# Skip installing istio as an add-on
initialize $@ --skip-istio-addon

TIMEOUT=10m
PROBE_TIMEOUT=20m
failed=0

header "Listing all the pods of the previous release"
wait_until_pods_running ${TEST_NAMESPACE}
wait_until_pods_running ${TEST_EVENTING_NAMESPACE}

header "Running preupgrade tests for Knative Serving"
# Go to the knative serving repo
cd ${KNATIVE_DIR}/serving
go_test_e2e -tags=preupgrade -timeout=${TIMEOUT} ./test/upgrade \
  --resolvabledomain="false" "--https" || fail_test=1

header "Starting prober test for serving"
# Remove this in case we failed to clean it up in an earlier test.
rm -f /tmp/prober-signal
rm -f /tmp/autoscaling-signal
rm -f /tmp/autoscaling-tbc-signal
go_test_e2e -tags=probe -timeout=${PROBE_TIMEOUT} ./test/upgrade \
  --resolvabledomain="false" "--https" &
PROBER_PID_SERVING=$!
echo "Prober PID Serving is ${PROBER_PID_SERVING}"

create_latest_custom_resource

# If we got this far, the operator installed Knative of the latest source code.
header "Running tests for Knative Operator"
# Run the postupgrade tests under operator
# Operator tests here will make sure that all the Knative deployments reach the desired states and operator CR is
# in ready state.
cd ${OPERATOR_DIR}
go_test_e2e -tags=postupgrade -timeout=${TIMEOUT} ./test/upgrade \
  --preservingversion="${PREVIOUS_SERVING_RELEASE_VERSION}" --preeventingversion="${PREVIOUS_EVENTING_RELEASE_VERSION}" || failed=1

header "Listing all the pods of the current release"
wait_until_pods_running ${TEST_NAMESPACE}
wait_until_pods_running ${TEST_EVENTING_NAMESPACE}

header "Running postupgrade tests for Knative Serving"
# Run the postupgrade tests under serving
cd ${KNATIVE_DIR}/serving
go_test_e2e -tags=postupgrade -timeout=${TIMEOUT} ./test/upgrade || failed=1

install_previous_knative

header "Running postdowngrade tests for Knative Operator"
cd ${OPERATOR_DIR}
go_test_e2e -tags=postdowngrade -timeout=${TIMEOUT} ./test/downgrade \
  --preservingversion="${PREVIOUS_SERVING_RELEASE_VERSION}" --preeventingversion="${PREVIOUS_EVENTING_RELEASE_VERSION}" || failed=1

header "Listing all the pods of the previous release"
wait_until_pods_running ${TEST_NAMESPACE}
wait_until_pods_running ${TEST_EVENTING_NAMESPACE}

header "Running postdowngrade tests for Knative Serving"
cd ${KNATIVE_DIR}/serving
go_test_e2e -tags=postdowngrade -timeout=${TIMEOUT} ./test/upgrade \
  --resolvabledomain="false" || fail_test

echo "done" > /tmp/prober-signal
echo "done" > /tmp/autoscaling-signal
echo "done" > /tmp/autoscaling-tbc-signal
header "Waiting for prober test for Knative Serving"
wait ${PROBER_PID_SERVING} || fail_test "Prober failed"

# Require that tests succeeded.
(( failed )) && fail_test

success
