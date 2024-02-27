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

source "$(dirname "${BASH_SOURCE[0]}")/e2e-common.sh"

export SERVING_TESTS_NAMESPACE="serving-tests"
export SERVING_TESTS_ALT_NAMESPACE="serving-tests-alt"

# Create test resources and images
function test_setup() {
  create_namespace
  download_knative "knative/serving" serving "${KNATIVE_REPO_BRANCH}"
  create_test_namespace_serving

  echo ">> Uploading test images..."
  # We only need to build and publish two images among all the test images
  ${OPERATOR_DIR}/test/upload-test-images.sh ${KNATIVE_DIR}/serving "test/test_images/pizzaplanetv1"
  ${OPERATOR_DIR}/test/upload-test-images.sh ${KNATIVE_DIR}/serving "test/test_images/pizzaplanetv2"
  ${OPERATOR_DIR}/test/upload-test-images.sh ${KNATIVE_DIR}/serving "test/test_images/autoscale"

  test_setup_logging

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

# Create test namespaces for serving
function create_test_namespace_serving() {
  kubectl get ns ${SERVING_TESTS_NAMESPACE} || kubectl create namespace ${SERVING_TESTS_NAMESPACE}
  kubectl get ns ${SERVING_TESTS_ALT_NAMESPACE} || kubectl create namespace ${SERVING_TESTS_ALT_NAMESPACE}
}

# Skip installing istio as an add-on.
initialize "$@" --skip-istio-addon

TIMEOUT=${TIMEOUT_CI}

header "Running upgrade tests"

go_test_e2e -tags=upgradeserving -timeout=${TIMEOUT} \
  ./test/upgrade \
  --preservingversion="${PREVIOUS_SERVING_RELEASE_VERSION}" --preeventingversion="${PREVIOUS_EVENTING_RELEASE_VERSION}" \
  || fail_test

# Remove the kail log file if the test flow passes.
# This is for preventing too many large log files to be uploaded to GCS in CI.
rm "${ARTIFACTS}/k8s.log-$(basename "${E2E_SCRIPT}").txt"
success
