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

# This script provides helper methods to perform cluster actions.
source $(dirname $0)/../vendor/knative.dev/test-infra/scripts/e2e-tests.sh

# The previous serving release, installed by the operator.
readonly PREVIOUS_SERVING_RELEASE_VERSION="0.17.4"
# The previous eventing release, installed by the operator.
readonly PREVIOUS_EVENTING_RELEASE_VERSION="0.17.9"
# This is the branch name of serving and eventing repo, where we run the upgrade tests.
readonly KNATIVE_REPO_BRANCH="release-0.18" #${PULL_BASE_REF}
# The istio branch version for Knative.
readonly KNATIVE_ISTIO_BRANCH_VERSION="0.16.0"
# Istio version we test with
readonly ISTIO_VERSION="1.5-latest"
# Test without Istio mesh enabled
readonly ISTIO_MESH=0
# Namespaces used for tests
export TEST_NAMESPACE="${TEST_NAMESPACE:-knative-operator-testing}"
export SYSTEM_NAMESPACE=${TEST_NAMESPACE}           # knative-serving
export TEST_EVENTING_NAMESPACE=${TEST_NAMESPACE}    # knative-eventing
export TEST_RESOURCE="knative"    # knative-eventing

# Boolean used to indicate whether to generate serving YAML based on the latest code in the branch KNATIVE_REPO_BRANCH.
GENERATE_SERVING_YAML=0

readonly OPERATOR_DIR=$(dirname $(cd "$(dirname "$0")" >/dev/null 2>&1 ; pwd -P))
readonly KNATIVE_DIR=$(dirname ${OPERATOR_DIR})
release_yaml="$(mktemp)"
release_eventing_yaml="$(mktemp)"

# Add function call to trap
# Parameters: $1 - Function to call
#             $2...$n - Signals for trap
function add_trap() {
  local cmd=$1
  shift
  for trap_signal in $@; do
    local current_trap="$(trap -p $trap_signal | cut -d\' -f2)"
    local new_cmd="($cmd)"
    [[ -n "${current_trap}" ]] && new_cmd="${current_trap};${new_cmd}"
    trap -- "${new_cmd}" $trap_signal
  done
}

# Setup and run kail in the background to collect logs
# from all pods.
function test_setup_logging() {
  echo ">> Setting up logging..."

  # Install kail if needed.
  if ! which kail > /dev/null; then
    bash <( curl -sfL https://raw.githubusercontent.com/boz/kail/master/godownloader.sh) -b "$GOPATH/bin"
  fi

  # Capture all logs.
  kail > ${ARTIFACTS}/k8s.log-$(basename ${E2E_SCRIPT}).txt &
  local kail_pid=$!
  # Clean up kail so it doesn't interfere with job shutting down
  add_trap "kill $kail_pid || true" EXIT
}

# Generic test setup. Used by the common test scripts.
function test_setup() {
  test_setup_logging
}

# Choose a correct istio-crds.yaml file.
# - $1 specifies Istio version.
function istio_crds_yaml() {
  local istio_version="$1"
  echo "third_party/${istio_version}/istio-crds.yaml"
}

# Choose a correct istio.yaml file.
# - $1 specifies Istio version.
# - $2 specifies whether we should use mesh.
function istio_yaml() {
  local istio_version="$1"
  local istio_mesh=$2
  local suffix=""
  if [[ $istio_mesh -eq 0 ]]; then
    suffix="ci-no-mesh"
  else
    suffix="ci-mesh"
  fi
  echo "third_party/${istio_version}/istio-${suffix}.yaml"
}

# Download the repository of Knative. The purpose of this function is to download the source code of
# knative component for further use, based on component name and branch name.
# Parameters:
#  $1 - component repo name, either knative/serving or knative/eventing,
#  $2 - component name,
#  $3 - branch of the repository.
function download_knative() {
  local component_repo component_name
  component_repo=$1
  component_name=$2
  # Go the directory to download the source code of knative
  cd ${KNATIVE_DIR}
  # Download the source code of knative
  git clone "https://github.com/${component_repo}.git" "${component_name}"
  cd "${component_name}"
  local branch=$3
  if [ -n "${branch}" ] ; then
    git fetch origin ${branch}:${branch}
    git checkout ${branch}
  fi
  cd ${OPERATOR_DIR}
}

# Install Istio.
function install_istio() {
  local base_url="https://raw.githubusercontent.com/knative/serving/v${KNATIVE_ISTIO_BRANCH_VERSION}"
  local istio_version="istio-${ISTIO_VERSION}"
  if [[ ${istio_version} == *-latest ]] ; then
    istio_version=$(curl https://raw.githubusercontent.com/knative/serving/v${KNATIVE_ISTIO_BRANCH_VERSION}/third_party/${istio_version})
  fi
  INSTALL_ISTIO_CRD_YAML="${base_url}/$(istio_crds_yaml $istio_version)"
  INSTALL_ISTIO_YAML="${base_url}/$(istio_yaml $istio_version $ISTIO_MESH)"

  echo ">> Installing Istio"
  echo "Istio CRD YAML: ${INSTALL_ISTIO_CRD_YAML}"
  echo "Istio YAML: ${INSTALL_ISTIO_YAML}"

  echo ">> Bringing up Istio"
  echo ">> Running Istio CRD installer"
  kubectl apply -f "${INSTALL_ISTIO_CRD_YAML}" || return 1

  echo ">> Running Istio"
  kubectl apply -f "${INSTALL_ISTIO_YAML}" || return 1
}

function create_namespace() {
  echo ">> Creating test namespaces"
  # All the custom resources and Knative Serving resources are created under this TEST_NAMESPACE.
  kubectl create namespace $TEST_NAMESPACE
  kubectl get ns $TEST_EVENTING_NAMESPACE || kubectl create ns $TEST_EVENTING_NAMESPACE
}

function install_operator() {
  cd ${OPERATOR_DIR}
  header "Installing Knative operator"
  # Deploy the operator
  ko apply -f config/
  wait_until_pods_running default || fail_test "Operator did not come up"
}

# Uninstalls Knative Serving from the current cluster.
function knative_teardown() {
  echo ">> Uninstalling Knative serving"
  echo "Istio YAML: ${INSTALL_ISTIO_YAML}"
  echo ">> Bringing down Serving"
  kubectl delete -n $TEST_NAMESPACE KnativeServing --all
  echo ">> Bringing down Eventing"
  kubectl delete -n $TEST_NAMESPACE KnativeEventing --all
  echo ">> Bringing down Istio"
  kubectl delete --ignore-not-found=true -f "${INSTALL_ISTIO_YAML}" || return 1
  kubectl delete --ignore-not-found=true clusterrolebinding cluster-admin-binding
  echo ">> Bringing down Operator"
  ko delete --ignore-not-found=true -f config/ || return 1
  echo ">> Removing test namespaces"
  kubectl delete all --all --ignore-not-found --now --timeout 60s -n $TEST_NAMESPACE
  kubectl delete --ignore-not-found --now --timeout 300s namespace $TEST_NAMESPACE
}

function wait_for_file() {
  local file timeout waits
  file="$1"
  waits=300
  timeout=$waits

  echo "Waiting for existence of file: ${file}"

  while [ ! -f "${file}" ]; do
    # When the timeout is equal to zero, show an error and leave the loop.
    if [ "${timeout}" == 0 ]; then
      echo "ERROR: Timeout (${waits}s) while waiting for the file ${file}."
      return 1
    fi

    sleep 1

    # Decrease the timeout of one
    ((timeout--))
  done
  return 0
}

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
