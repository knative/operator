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

readonly LATEST_EVENTING_OPERATOR_RELEASE_VERSION="v0.13.2"
readonly LATEST_EVENTING_RELEASE_VERSION="v0.13.4"

export GO111MODULE=auto

source $(dirname $0)/e2e-common.sh

function install_previous_serving_operator_release() {
  local full_url="https://github.com/knative/serving-operator/releases/download/${LATEST_SERVING_OPERATOR_RELEASE_VERSION}/serving-operator.yaml"

  wget "${full_url}" -O "${release_yaml}" \
      || fail_test "Unable to download latest Knative Serving Operator release."

  donwload_knative_serving ${SERVING_REPO_BRANCH}
  install_istio || fail_test "Istio installation failed"
  install_previous_serving_release
}

function install_previous_serving_release() {
  header "Installing Knative Serving operator previous public release"
  kubectl apply -f "${release_yaml}" || fail_test "Knative Serving Operator latest release installation failed"
  wait_until_pods_running default || fail_test "Serving Operator did not come up"
}

function create_custom_resource() {
  echo ">> Creating the custom resource of Knative Serving:"
  cat <<EOF | kubectl apply -f -
apiVersion: operator.knative.dev/v1alpha1
kind: KnativeServing
metadata:
  name: knative-serving
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
  name: knative-eventing
  namespace: ${TEST_EVENTING_NAMESPACE}
EOF
}

function install_previous_eventing_operator_release() {
  local full_url="https://github.com/knative/eventing-operator/releases/download/${LATEST_EVENTING_OPERATOR_RELEASE_VERSION}/eventing-operator.yaml"

  wget "${full_url}" -O "${release_eventing_yaml}" \
      || fail_test "Unable to download latest Knative Eventing Operator release."

  install_previous_eventing_release
}

function install_previous_eventing_release() {
  header "Installing Knative Eventing operator previous public release"
  kubectl apply -f "${release_eventing_yaml}" || fail_test "Knative Eventing Operator latest release installation failed"
  wait_until_pods_running default || fail_test "Eventing Operator did not come up"
}

function knative_setup() {
  create_namespace
  install_previous_serving_operator_release
  install_previous_eventing_operator_release
  create_custom_resource
  wait_until_pods_running ${TEST_NAMESPACE}
  wait_until_pods_running ${TEST_EVENTING_NAMESPACE}
}

# Create test resources and images
function test_setup() {
  if (( GENERATE_SERVING_YAML )); then
    generate_latest_serving_manifest ${SERVING_REPO_BRANCH}
  fi
  echo ">> Creating test resources (test/config/) in Knative Serving repository"
  cd ${KNATIVE_SERVING_DIR}/serving
  ko apply ${KO_FLAGS} -f test/config/ || return 1

  echo ">> Uploading test images..."
  # We only need to build and publish two images among all the test images
  ${OPERATOR_DIR}/test/upload-test-images.sh ${KNATIVE_SERVING_DIR}/serving "test/test_images/pizzaplanetv1"
  ${OPERATOR_DIR}/test/upload-test-images.sh ${KNATIVE_SERVING_DIR}/serving "test/test_images/pizzaplanetv2"

  echo ">> Waiting for Ingress provider to be running..."
  if [[ -n "${ISTIO_VERSION}" ]]; then
    wait_until_pods_running istio-system || return 1
    wait_until_service_has_external_ip istio-system istio-ingressgateway
  fi
  cd ${OPERATOR_DIR}
}

# This function either generate the manifest based on a branch or download the latest manifest for Knative Serving.
# Parameter: $1 - branch name. If it is empty, download the manifest from nightly build.
function generate_latest_serving_manifest() {
  cd ${KNATIVE_SERVING_DIR}/serving
  mkdir -p output
  local branch=$1
  export YAML_OUTPUT_DIR=${KNATIVE_SERVING_DIR}/serving/output
  SERVING_YAML=${YAML_OUTPUT_DIR}/serving.yaml
  if [[ -n "${branch}" ]]; then
    git checkout ${branch}
    COMMIT_ID=$(git rev-parse --verify HEAD)
    echo ">> The latest commit ID of Knative Serving is ${COMMIT_ID}."
    # Generate the manifest
    export YAML_OUTPUT_DIR=${KNATIVE_SERVING_DIR}/serving/output
    ./hack/generate-yamls.sh ${KNATIVE_SERVING_DIR}/serving ${YAML_OUTPUT_DIR}/output.yaml
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
wait_until_pods_running ${TEST_EVENTING_NAMESPACE}

header "Running preupgrade tests"

cd ${KNATIVE_SERVING_DIR}/serving
go_test_e2e -tags=preupgrade -timeout=${TIMEOUT} ./test/upgrade \
  --resolvabledomain="false" "--https" || fail_test

# Remove this in case we failed to clean it up in an earlier test.
rm -f /tmp/prober-signal

go_test_e2e -tags=probe -timeout=${TIMEOUT} ./test/upgrade \
  --resolvabledomain="false" "--https" &
PROBER_PID=$!
echo "Prober PID is ${PROBER_PID}"

install_operator

# If we got this far, the operator installed Knative Serving of the latest source code.
header "Running tests for Knative Serving Operator"
failed=0

# Run the postupgrade tests under operator
# Operator tests here will make sure that all the Knative deployments reach the desired states and operator CR is
# in ready state.
cd ${OPERATOR_DIR}
go_test_e2e -tags=postupgrade -timeout=${TIMEOUT} ./test/upgrade || failed=1
wait_until_pods_running ${TEST_NAMESPACE}
wait_until_pods_running ${TEST_EVENTING_NAMESPACE}

header "Running tests under Knative Serving"
# Run the postupgrade tests under serving
cd ${KNATIVE_SERVING_DIR}/serving
go_test_e2e -tags=postupgrade -timeout=${TIMEOUT} ./test/upgrade || failed=1

# Verify with the bash script to make sure there is no resource with the label of the previous release.
list_resources="deployment,pod,service,apiservice,cm,crd,sa,ClusterRole,ClusterRoleBinding,Image,ValidatingWebhookConfiguration,\
MutatingWebhookConfiguration,Secret,RoleBinding,APIService,Gateway"
result="$(kubectl get ${list_resources} -l serving.knative.dev/release=${LATEST_SERVING_RELEASE_VERSION} --all-namespaces 2>/dev/null)"

# If the ${result} is not empty, we fail the tests, because the resources from the previous release still exist.
if [[ ! -z ${result} ]] ; then
  header "The following obsolete resources still exist for serving operator:"
  echo "${result}"
  fail_test "The resources with the label of previous release have not been removed."
fi

# Verify with the bash script to make sure there is no resource with the label of the previous release.
list_resources="deployment,pod,service,cm,crd,sa,ClusterRole,ClusterRoleBinding,ValidatingWebhookConfiguration,\
MutatingWebhookConfiguration,Secret,RoleBinding"
result="$(kubectl get ${list_resources} -l eventing.knative.dev/release=${LATEST_EVENTING_RELEASE_VERSION} --all-namespaces 2>/dev/null)"

# If the ${result} is not empty, we fail the tests, because the resources from the previous release still exist.
if [[ ! -z ${result} ]] ; then
  header "The following obsolete resources still exist for eventing operator:"
  echo "${result}"
  fail_test "The resources with the label of previous release have not been removed."
fi

install_previous_serving_release
install_previous_eventing_release
wait_until_pods_running ${TEST_NAMESPACE}
wait_until_pods_running ${TEST_EVENTING_NAMESPACE}

header "Running postdowngrade tests"
go_test_e2e -tags=postdowngrade -timeout=${TIMEOUT} ./test/upgrade \
  --resolvabledomain="false" || fail_test

echo "done" > /tmp/prober-signal

header "Waiting for prober test"
wait ${PROBER_PID} || fail_test "Prober failed"

# Require that tests succeeded.
(( failed )) && fail_test

success
