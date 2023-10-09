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
source "$(dirname "${BASH_SOURCE[0]}")/../vendor/knative.dev/hack/e2e-tests.sh"

# The previous serving release, installed by the operator. This value should be in the semantic format of major.minor.
readonly PREVIOUS_SERVING_RELEASE_VERSION="1.10"
# The previous eventing release, installed by the operator. This value should be in the semantic format of major.minor.
readonly PREVIOUS_EVENTING_RELEASE_VERSION="1.10"
# The target serving/eventing release to upgrade, installed by the operator. It can be a release available under
# kodata or an incoming new release. This value should be in the semantic format of major.minor.
readonly TARGET_RELEASE_VERSION="1.11"
# This is the branch name of knative repos, where we run the upgrade tests.
readonly KNATIVE_REPO_BRANCH="release-1.11"
# Namespaces used for tests
# This environment variable TEST_NAMESPACE defines the namespace to install Knative Serving.
export TEST_NAMESPACE="${TEST_NAMESPACE:-knative-operator-testing}"
export SYSTEM_NAMESPACE=${TEST_NAMESPACE}
# This environment variable TEST_EVENTING_NAMESPACE defines the namespace to install Knative Eventing.
# It is different from the namespace to install Knative Serving.
# We will use only one namespace, when Knative supports both components can coexist under one namespace.
export TEST_EVENTING_NAMESPACE="knative-eventing"
export TEST_RESOURCE="knative"
export TEST_EVENTING_MONITORING_NAMESPACE="knative-monitoring"
export KO_FLAGS="${KO_FLAGS:-}"
export INGRESS_CLASS=${INGRESS_CLASS:-istio.ingress.networking.knative.dev}

# Boolean used to indicate whether to generate serving YAML based on the latest code in the branch KNATIVE_SERVING_REPO_BRANCH.
GENERATE_SERVING_YAML=0

readonly OPERATOR_DIR="$(dirname "${BASH_SOURCE[0]}")/.."
readonly KNATIVE_DIR=$(dirname ${OPERATOR_DIR})
release_yaml="$(mktemp)"
release_eventing_yaml="$(mktemp)"

readonly SERVING_ARTIFACTS=("serving" "serving-crds.yaml" "serving-core.yaml" "serving-hpa.yaml" "serving-post-install-jobs.yaml")
readonly EVENTING_ARTIFACTS=("eventing" "eventing-crds.yaml" "eventing-core.yaml" "in-memory-channel.yaml" "mt-channel-broker.yaml"
  "eventing-post-install.yaml" "eventing-tls-networking.yaml")

function is_ingress_class() {
  [[ "${INGRESS_CLASS}" == *"${1}"* ]]
}

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
  echo ">> Installing Istio"
  curl -sL https://istio.io/downloadIstioctl | sh -
  $HOME/.istioctl/bin/istioctl install -y
}

function create_namespace() {
  echo ">> Creating test namespaces for knative serving and eventing"
  # All the custom resources and Knative Serving resources are created under this TEST_NAMESPACE.
  kubectl get ns ${TEST_NAMESPACE} || kubectl create namespace ${TEST_NAMESPACE}
  kubectl get ns ${TEST_EVENTING_NAMESPACE} || kubectl create namespace ${TEST_EVENTING_NAMESPACE}
}

function download_latest_release() {
  download_nightly_artifacts "${SERVING_ARTIFACTS[@]}"
  download_nightly_artifacts "${EVENTING_ARTIFACTS[@]}"
}

function download_nightly_artifacts() {
  array=("$@")
  component=${array[0]}
  unset array[0]
  counter=0
  linkprefix="https://storage.googleapis.com/knative-nightly/${component}/latest"
  version_exists=$(if_version_exists ${TARGET_RELEASE_VERSION} "knative-${component}")
  if [ "${version_exists}" == "no" ]; then
    header "Download the nightly build as the target version for Knative ${component}"
    knative_version_dir=${OPERATOR_DIR}/cmd/operator/kodata/knative-${component}/${TARGET_RELEASE_VERSION}
    mkdir ${knative_version_dir}
    for artifact in "${array[@]}";
      do
        ((counter=counter+1))
        wget ${linkprefix}/${artifact} -O ${knative_version_dir}/${counter}-${artifact}
      done
    if [ "${component}" == "serving" ]; then
      # Download the latest net-istio into the ingress directory.
      ingress_version_dir=${OPERATOR_DIR}/cmd/operator/kodata/ingress/${TARGET_RELEASE_VERSION}/istio
      mkdir -p ${ingress_version_dir}
      wget https://storage.googleapis.com/knative-nightly/net-istio/latest/net-istio.yaml -O ${ingress_version_dir}/net-istio.yaml
    fi
  fi
}

function install_operator() {
  create_namespace
  if is_ingress_class istio; then
    install_istio || fail_test "Istio installation failed"
  fi
  cd ${OPERATOR_DIR}
  download_latest_release
  header "Installing Knative operator"
  # Deploy the operator
  ko apply ${KO_FLAGS} -f config/
  wait_until_pods_running default || fail_test "Operator did not come up"
}

# Uninstalls Knative Serving from the current cluster.
function knative_teardown() {
  echo ">> Uninstalling Knative serving"
  echo ">> Bringing down Serving"
  kubectl delete -n $TEST_NAMESPACE KnativeServing --all
  echo ">> Bringing down Eventing"
  kubectl delete -n $TEST_NAMESPACE KnativeEventing --all
  echo ">> Bringing down Istio"
  $HOME/.istioctl/bin/istioctl x uninstall --purge
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
apiVersion: operator.knative.dev/v1beta1
kind: KnativeServing
metadata:
  name: ${TEST_RESOURCE}
  namespace: ${TEST_NAMESPACE}
spec:
  version: "${version}"
  config:
    domain:
      example.com: |
    tracing:
      backend: "zipkin"
      zipkin-endpoint: "http://zipkin.${TEST_EVENTING_MONITORING_NAMESPACE}.svc:9411/api/v2/spans"
      debug: "true"
      sample-rate: "1.0"
EOF
}

function create_knative_eventing() {
  version=${1}
  echo ">> Creating the custom resource of Knative Eventing:"
  cat <<-EOF | kubectl apply -f -
apiVersion: operator.knative.dev/v1beta1
kind: KnativeEventing
metadata:
  name: ${TEST_RESOURCE}
  namespace: ${TEST_EVENTING_NAMESPACE}
spec:
  version: "${version}"
  config:
    tracing:
      backend: "zipkin"
      zipkin-endpoint: "http://zipkin.${TEST_EVENTING_MONITORING_NAMESPACE}.svc:9411/api/v2/spans"
      debug: "true"
      sample-rate: "1.0"
EOF
}

function create_latest_custom_resource() {
  echo ">> Creating the custom resource of Knative Serving:"
  cat <<-EOF | kubectl apply -f -
apiVersion: operator.knative.dev/v1beta1
kind: KnativeServing
metadata:
  name: ${TEST_RESOURCE}
  namespace: ${TEST_NAMESPACE}
spec:
  version: "${TARGET_RELEASE_VERSION}"
  config:
    domain:
      example.com: |
    tracing:
      backend: "zipkin"
      zipkin-endpoint: "http://zipkin.${TEST_EVENTING_MONITORING_NAMESPACE}.svc:9411/api/v2/spans"
      debug: "true"
      sample-rate: "1.0"
EOF

  echo ">> Creating the custom resource of Knative Eventing:"
  cat <<-EOF | kubectl apply -f -
apiVersion: operator.knative.dev/v1beta1
kind: KnativeEventing
metadata:
  name: ${TEST_RESOURCE}
  namespace: ${TEST_EVENTING_NAMESPACE}
spec:
  version: "${TARGET_RELEASE_VERSION}"
  config:
    tracing:
      backend: "zipkin"
      zipkin-endpoint: "http://zipkin.${TEST_EVENTING_MONITORING_NAMESPACE}.svc:9411/api/v2/spans"
      debug: "true"
      sample-rate: "1.0"
EOF
}

function if_version_exists() {
  version=$1
  component=$2
  knative_dir=${OPERATOR_DIR}/cmd/operator/kodata/${component}
  versions=$(ls ${knative_dir})
  for eachversion in ${versions}
  do
    if [[ "${eachversion}" == ${version}* ]]; then
      echo "yes"
      exit
    fi
  done
  echo "no"
}
