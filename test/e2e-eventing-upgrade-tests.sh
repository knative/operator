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

export GO111MODULE=on

source "$(dirname "${BASH_SOURCE[0]}")/e2e-common.sh"

# The environment variable EVENTING_UPGRADE_TESTS_SERVING_USE controls the usage of ksvc forwarder of Serving
export EVENTING_UPGRADE_TESTS_SERVING_USE=false
# The environment variable EVENTING_UPGRADE_TESTS_SERVING_SCALETOZERO controls whether the ksvc can scale to zero.
# FIXME(ksuszyns): remove when knative/operator#297 is fixed
export EVENTING_UPGRADE_TESTS_SERVING_SCALETOZERO=false
# Installs Zipkin for tracing tests.
readonly KNATIVE_EVENTING_MONITORING_YAML="test/config/monitoring.yaml"

TMP_DIR=$(mktemp -d -t "ci-$(date +%Y-%m-%d-%H-%M-%S)-XXXXXXXXXX")
readonly TMP_DIR

# Create test resources and images
function test_setup() {
  download_knative "knative/eventing" eventing "${KNATIVE_REPO_BRANCH}"
  # Install kail if needed.
  if ! which kail >/dev/null; then
    bash <(curl -sfL https://raw.githubusercontent.com/boz/kail/master/godownloader.sh) -b "$GOPATH/bin"
  fi

  # Capture all logs.
  kail >${ARTIFACTS}/k8s.log.txt &
  local kail_pid=$!
  # Clean up kail so it doesn't interfere with job shutting down
  add_trap "kill $kail_pid || true" EXIT

  echo ">> Publish test images for eventing"
  ${OPERATOR_DIR}/test/upload-test-images.sh ${KNATIVE_DIR}/eventing "test/test_images"

  # Ensure knative monitoring is installed only once
  cd ${KNATIVE_DIR}/eventing
  kubectl get ns ${TEST_EVENTING_MONITORING_NAMESPACE}|| kubectl create namespace ${TEST_EVENTING_MONITORING_NAMESPACE}
  knative_monitoring_pods=$(kubectl get pods -n ${TEST_EVENTING_MONITORING_NAMESPACE} \
    --field-selector status.phase=Running 2> /dev/null | tail -n +2 | wc -l)
  if ! [[ ${knative_monitoring_pods} -gt 0 ]]; then
    echo ">> Installing Knative Monitoring"
    echo "Installing Monitoring from ${KNATIVE_EVENTING_MONITORING_YAML}"
    local KNATIVE_EVENTING_MONITORING_NAME=${TMP_DIR}/${KNATIVE_EVENTING_MONITORING_YAML##*/}
    sed "s/namespace: ${TEST_EVENTING_NAMESPACE}/namespace: ${TEST_EVENTING_MONITORING_NAMESPACE}/g" ${KNATIVE_EVENTING_MONITORING_YAML} > ${KNATIVE_EVENTING_MONITORING_NAME}
    kubectl apply -f "${KNATIVE_EVENTING_MONITORING_NAME}"
    wait_until_pods_running ${TEST_EVENTING_MONITORING_NAMESPACE}
  else
    echo ">> Knative Monitoring seems to be running, pods running: ${knative_monitoring_pods}."
  fi

  cd ${OPERATOR_DIR}
}

# Skip installing istio as an add-on.
initialize "$@"

TIMEOUT=${TIMEOUT_CI}

header "Running upgrade tests"

go_test_e2e -tags=upgradeeventing -timeout=${TIMEOUT} \
  ./test/upgrade \
  --preservingversion="${PREVIOUS_SERVING_RELEASE_VERSION}" --preeventingversion="${PREVIOUS_EVENTING_RELEASE_VERSION}" \
  || fail_test

success
