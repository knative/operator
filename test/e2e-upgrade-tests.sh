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

readonly EVENTING_READY_FILE="/tmp/prober-ready-eventing"
readonly EVENTING_PROBER_FILE="/tmp/prober-signal-eventing"

function download_install_previous_operator_release() {
  local full_url="https://github.com/knative/operator/releases/download/v${PREVIOUS_OPERATOR_RELEASE_VERSION}/operator.yaml"

  wget "${full_url}" -O "${release_yaml}" \
      || fail_test "Unable to download latest Knative Operator release."

  install_istio || fail_test "Istio installation failed"
  install_previous_operator_release
}

function install_previous_operator_release() {
  header "Installing Knative operator of the previous public release"
  kubectl apply -f "${release_yaml}" || fail_test "Knative Operator previous release installation failed"
  wait_until_pods_running default || fail_test "Operator did not come up"
}

function create_custom_resource() {
  echo ">> Creating the custom resource of Knative Serving:"
  cat <<EOF | kubectl apply -f -
apiVersion: operator.knative.dev/v1alpha1
kind: KnativeServing
metadata:
  name: ${TEST_RESOURCE}
  namespace: ${TEST_NAMESPACE}
spec:
  config:
    defaults:
      revision-timeout-seconds: "300"  # 5 minutes
    autoscaler:
      stable-window: "60s"
    deployment:
      registriesSkippingTagResolving: "ko.local,dev.local"
    logging:
      loglevel.controller: "debug"
EOF
  echo ">> Creating the custom resource of Knative Eventing:"
  cat <<-EOF | kubectl apply -f -
apiVersion: operator.knative.dev/v1alpha1
kind: KnativeEventing
metadata:
  name: ${TEST_RESOURCE}
  namespace: ${TEST_NAMESPACE}
EOF
}

function knative_setup() {
  create_namespace
  download_install_previous_operator_release
  donwload_knative "serving" ${KNATIVE_REPO_BRANCH}
  donwload_knative "eventing" ${KNATIVE_REPO_BRANCH}
  create_custom_resource
  wait_until_pods_running ${TEST_NAMESPACE}
}

# Create test resources and images
function test_setup() {
  if (( GENERATE_SERVING_YAML )); then
    generate_latest_serving_manifest ${KNATIVE_REPO_BRANCH}
  fi
  echo ">> Creating test resources (test/config/) in Knative Serving repository"
  cd ${KNATIVE_DIR}/serving
  local SERVING_TEST_CONFIG_DIR=${TMP_DIR}/serving/test/config
  mkdir -p ${SERVING_TEST_CONFIG_DIR}
  cp -r test/config/* ${SERVING_TEST_CONFIG_DIR}
  find ${SERVING_TEST_CONFIG_DIR} -type f -name "*.yaml" -exec sed -i "s/${KNATIVE_DEFAULT_NAMESPACE}/${TEST_NAMESPACE}/g" {} +

  ko apply ${KO_FLAGS} -f ${SERVING_TEST_CONFIG_DIR} || return 1

  echo ">> Uploading test images..."
  # We only need to build and publish two images among all the test images
  ${OPERATOR_DIR}/test/upload-test-images.sh ${KNATIVE_DIR}/serving "test/test_images/pizzaplanetv1"
  ${OPERATOR_DIR}/test/upload-test-images.sh ${KNATIVE_DIR}/serving "test/test_images/pizzaplanetv2"

  test_setup_logging

  echo ">> Waiting for Ingress provider to be running..."
  if [[ -n "${ISTIO_VERSION}" ]]; then
    wait_until_pods_running istio-system || return 1
    wait_until_service_has_external_ip istio-system istio-ingressgateway
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

  echo ">> Publish test images for eventing"
  ${OPERATOR_DIR}/test/upload-test-images.sh ${KNATIVE_DIR}/eventing "test/test_images"

  cd ${OPERATOR_DIR}
}

# This function either generate the manifest based on a branch or download the latest manifest for Knative Serving.
# Parameter: $1 - branch name. If it is empty, download the manifest from nightly build.
function generate_latest_serving_manifest() {
  cd ${KNATIVE_DIR}/serving
  mkdir -p output
  local branch=$1
  export YAML_OUTPUT_DIR=${KNATIVE_DIR}/serving/output
  SERVING_YAML=${YAML_OUTPUT_DIR}/serving.yaml
  if [[ -n "${branch}" ]]; then
    git checkout ${branch}
    COMMIT_ID=$(git rev-parse --verify HEAD)
    echo ">> The latest commit ID of Knative Serving is ${COMMIT_ID}."
    # Generate the manifest
    export YAML_OUTPUT_DIR=${KNATIVE_DIR}/serving/output
    ./hack/generate-yamls.sh ${KNATIVE_DIR}/serving ${YAML_OUTPUT_DIR}/output.yaml
  else
    echo ">> Download the latest nightly build of Knative Serving."
    # Download the latest manifest
    wget -O ${SERVING_YAML} https://storage.googleapis.com/knative-nightly/serving/latest/serving.yaml
  fi

  if [[ -f "${SERVING_YAML}" ]]; then
    echo ">> Replacing the current manifest in operator with the generated manifest"
    rm -rf ${OPERATOR_DIR}/cmd/serving-operator/kodata/knative-serving/*
    cp ${SERVING_YAML} ${OPERATOR_DIR}/cmd/serving-operator/kodata/knative-serving/serving.yaml
  else
    echo ">> The serving.yaml was not generated, so keep the current manifest"
  fi

  # Go back to the directory of operator
  cd ${OPERATOR_DIR}
}

# Skip installing istio as an add-on
initialize $@ --skip-istio-addon

TIMEOUT=20m

header "Running preupgrade tests"
go_test_e2e -tags=preupgrade -timeout=${TIMEOUT} ./test/upgrade || fail_test

header "Listing all the pods of the previous release"
wait_until_pods_running ${TEST_NAMESPACE}

header "Running preupgrade tests"

# Go to the knative serving repo
cd ${KNATIVE_DIR}/serving
go_test_e2e -tags=preupgrade -timeout=${TIMEOUT} ./test/upgrade \
  --resolvabledomain="false" "--https" || fail_test

header "Starting prober test for serving"
# Remove this in case we failed to clean it up in an earlier test.
rm -f /tmp/prober-signal
go_test_e2e -tags=probe -timeout=${TIMEOUT} ./test/upgrade \
  --resolvabledomain="false" "--https" &
PROBER_PID_SERVING=$!
echo "Prober PID Serving is ${PROBER_PID_SERVING}"

# Go to the knative eventing repo
cd ${KNATIVE_DIR}/eventing
echo "check env var"
set
go_test_e2e -tags=preupgrade -timeout="${TIMEOUT}" ./test/upgrade || fail_test

header "Starting prober test for eventing"
# Remove this in case we failed to clean it up in an earlier test.
rm -f ${EVENTING_PROBER_FILE}
go_test_e2e -tags=probe -timeout="${TIMEOUT}" ./test/upgrade --pipefile="${EVENTING_PROBER_FILE}" --readyfile="${EVENTING_READY_FILE}" &
PROBER_PID_EVENTING=$!
echo "Prober PID Serving is ${PROBER_PID_EVENTING}"

install_operator

# If we got this far, the operator installed Knative of the latest source code.
header "Running tests for Knative Operator"
failed=0

# Run the postupgrade tests under operator
# Operator tests here will make sure that all the Knative deployments reach the desired states and operator CR is
# in ready state.
cd ${OPERATOR_DIR}
go_test_e2e -tags=postupgrade -timeout=${TIMEOUT} ./test/upgrade \
  --preservingversion="${PREVIOUS_SERVING_RELEASE_VERSION}" --preeventingversion="${PREVIOUS_EVENTING_RELEASE_VERSION}" || failed=1
wait_until_pods_running ${TEST_NAMESPACE}

header "Running postupgrade tests for Knative Serving"
# Run the postupgrade tests under serving
cd ${KNATIVE_DIR}/serving
go_test_e2e -tags=postupgrade -timeout=${TIMEOUT} ./test/upgrade || failed=1

header "Running postupgrade tests for Knative Eventing"
cd ${KNATIVE_DIR}/eventing
go_test_e2e -tags=postupgrade -timeout="${TIMEOUT}" ./test/upgrade || fail_test

install_previous_operator_release
wait_until_pods_running ${TEST_NAMESPACE}

header "Running postdowngrade tests for Knative Serving"
cd ${KNATIVE_DIR}/serving
go_test_e2e -tags=postdowngrade -timeout=${TIMEOUT} ./test/upgrade \
  --resolvabledomain="false" || fail_test

header "Running postdowngrade tests for Knative Eventing"
cd ${KNATIVE_DIR}/eventing
go_test_e2e -tags=postdowngrade -timeout=${TIMEOUT} ./test/upgrade || fail_test

echo "done" > /tmp/prober-signal

header "Waiting for prober test for Knative Serving"
wait ${PROBER_PID_SERVING} || fail_test "Prober failed"

echo "done" > ${EVENTING_PROBER_FILE}
header "Waiting for prober test for Knative Eventing"
wait ${PROBER_PID_EVENTING} || fail_test "Prober failed"

# Require that tests succeeded.
(( failed )) && fail_test

success
