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
readonly PREVIOUS_SERVING_RELEASE_VERSION="1.21"
# The previous eventing release, installed by the operator. This value should be in the semantic format of major.minor.
readonly PREVIOUS_EVENTING_RELEASE_VERSION="1.21"
# The target serving/eventing release to upgrade, installed by the operator. It can be a release available under
# kodata or an incoming new release. This value should be in the semantic format of major.minor.
readonly TARGET_RELEASE_VERSION="latest"
# This is the branch name of knative repos, where we run the upgrade tests.
# Default to empty for local runs (not in Prow).
readonly KNATIVE_REPO_BRANCH="${PULL_BASE_REF:-}"
# Namespaces used for tests
# This environment variable TEST_NAMESPACE defines the namespace to install Knative Serving.
export TEST_NAMESPACE="${TEST_NAMESPACE:-knative-operator-testing}"
export SYSTEM_NAMESPACE=${TEST_NAMESPACE}
export TEST_OPERATOR_NAMESPACE="knative-operator"
# This environment variable TEST_EVENTING_NAMESPACE defines the namespace to install Knative Eventing.
# It is different from the namespace to install Knative Serving.
# We will use only one namespace, when Knative supports both components can coexist under one namespace.
export TEST_EVENTING_NAMESPACE="knative-eventing"
export TEST_RESOURCE="knative"
export TEST_EVENTING_MONITORING_NAMESPACE="knative-monitoring"
export KO_FLAGS="${KO_FLAGS:-}"
export INGRESS_CLASS=${INGRESS_CLASS:-istio.ingress.networking.knative.dev}
export TIMEOUT_CI=30m

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
  for trap_signal in "$@"; do
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
  if ! command -v kail > /dev/null; then
    bash <(curl -sfL https://raw.githubusercontent.com/boz/kail/master/godownloader.sh) -b "$GOPATH/bin"
  fi

  # Capture all logs.
  kail > "${ARTIFACTS}/k8s.log-$(basename "${E2E_SCRIPT}").txt" &
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
  curl -sL https://istio.io/downloadIstioctl | ISTIO_VERSION=1.25.2 sh -
  local istioctl_args=(install -y)
  if [[ "${CLOUD_PROVIDER:-gke}" == "gke" ]]; then
    istioctl_args+=(--set values.cni.cniBinDir=/home/kubernetes/bin)
  fi
  $HOME/.istioctl/bin/istioctl "${istioctl_args[@]}"
}

function create_namespace() {
  echo ">> Creating test namespaces for knative operator"
  kubectl get ns ${TEST_OPERATOR_NAMESPACE} || kubectl create namespace ${TEST_OPERATOR_NAMESPACE}
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
  wait_until_pods_running ${TEST_OPERATOR_NAMESPACE} || fail_test "Operator did not come up"
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

# Multi-cluster e2e helpers

function create_spoke_cluster() {
  echo ">> Creating spoke KinD cluster: ${SPOKE_CLUSTER_NAME}"
  if kind get clusters 2>/dev/null | grep -q "^${SPOKE_CLUSTER_NAME}$"; then
    echo ">> Spoke cluster already exists, reusing"
  else
    kind create cluster --name "${SPOKE_CLUSTER_NAME}" --kubeconfig "${SPOKE_HOST_KUBECONFIG}" --wait 120s || return 1
  fi
  # internal kubeconfig for hub->spoke access via docker bridge
  kind get kubeconfig --internal --name "${SPOKE_CLUSTER_NAME}" > "${SPOKE_KUBECONFIG}" || return 1
  kind get kubeconfig --name "${SPOKE_CLUSTER_NAME}" > "${SPOKE_HOST_KUBECONFIG}" || return 1
  export SPOKE_KUBECONFIG SPOKE_HOST_KUBECONFIG
  echo ">> Spoke kubeconfig ready"

  echo ">> Waiting for spoke nodes and core components"
  KUBECONFIG="${SPOKE_HOST_KUBECONFIG}" kubectl wait --for=condition=Ready node --all --timeout=120s || return 1
  KUBECONFIG="${SPOKE_HOST_KUBECONFIG}" kubectl -n kube-system rollout status deployment/coredns --timeout=120s || return 1
}

function delete_spoke_cluster() {
  if kind get clusters 2>/dev/null | grep -q "^${SPOKE_CLUSTER_NAME}$"; then
    kind delete cluster --name "${SPOKE_CLUSTER_NAME}" --kubeconfig "${SPOKE_HOST_KUBECONFIG}"
  fi
  rm -f "${SPOKE_KUBECONFIG}" "${SPOKE_HOST_KUBECONFIG}"
}

function dump_spoke_state() {
  if [[ -z "${SPOKE_HOST_KUBECONFIG:-}" || ! -f "${SPOKE_HOST_KUBECONFIG}" ]]; then
    echo ">> [dump_spoke_state] SPOKE_HOST_KUBECONFIG not present, skipping"
    return 0
  fi
  local out="${ARTIFACTS:-/tmp}/spoke-dump.txt"
  local -a kc=(kubectl --request-timeout=30s)
  echo ">> Dumping spoke cluster state to ${out}"
  {
    echo "=== kubectl get nodes ==="
    KUBECONFIG="${SPOKE_HOST_KUBECONFIG}" "${kc[@]}" get nodes -o wide || true
    echo
    echo "=== kubectl get deployments,statefulsets,daemonsets,pods,svc,cm,secret -A --show-kind ==="
    KUBECONFIG="${SPOKE_HOST_KUBECONFIG}" "${kc[@]}" \
      get deployments,statefulsets,daemonsets,pods,svc,cm,secret -A --show-kind -o wide || true
    echo
    echo "=== kubectl get events -A ==="
    KUBECONFIG="${SPOKE_HOST_KUBECONFIG}" "${kc[@]}" get events -A --sort-by=.lastTimestamp || true
    echo
    local spoke_ns="${TEST_NAMESPACE:-knative-operator-testing}"
    echo "=== pod logs in namespace ${spoke_ns} ==="
    KUBECONFIG="${SPOKE_HOST_KUBECONFIG}" "${kc[@]}" -n "${spoke_ns}" \
      logs --all-containers=true --prefix --tail=200 -l "app" 2>/dev/null || true
  } > "${out}" 2>&1 || true
}

function dump_hub_state() {
  local out="${ARTIFACTS:-/tmp}/hub-dump.txt"
  local -a kc=(kubectl --request-timeout=30s)
  local op_ns="${TEST_OPERATOR_NAMESPACE:-knative-operator}"
  echo ">> Dumping hub cluster state to ${out}"
  {
    echo "=== kubectl -n ${op_ns} get deploy,po,cm,secret ==="
    "${kc[@]}" -n "${op_ns}" get deploy,po,cm,secret -o wide || true
    echo
    echo "=== kubectl -n ${op_ns} describe deployment/knative-operator ==="
    "${kc[@]}" -n "${op_ns}" describe deployment/knative-operator || true
    echo
    echo "=== kubectl -n ${op_ns} logs deployment/knative-operator --all-containers=true --tail=2000 ==="
    "${kc[@]}" -n "${op_ns}" logs deployment/knative-operator \
      --all-containers=true --tail=2000 || true
    echo
    echo "=== kubectl get clusterprofile -A -o yaml ==="
    "${kc[@]}" get clusterprofile -A -o yaml || true
    echo
    echo "=== kubectl get knativeserving,knativeeventing -A -o yaml ==="
    "${kc[@]}" get knativeserving,knativeeventing -A -o yaml || true
    echo
    echo "=== kubectl get events -A --sort-by=.lastTimestamp ==="
    "${kc[@]}" get events -A --sort-by=.lastTimestamp || true
  } > "${out}" 2>&1 || true
}

function install_cluster_inventory_crd() {
  echo ">> Installing ClusterProfile CRD on hub"
  kubectl apply -f "${CLUSTER_INVENTORY_CRD_URL}" || return 1
  kubectl wait --for=condition=Established --timeout=60s \
    crd/clusterprofiles.multicluster.x-k8s.io || return 1
}

function _spoke_bootstrap_token() {
  local out_file="$1"
  local sa_ns="kube-system"
  local sa_name="knative-operator-e2e"
  KUBECONFIG="${SPOKE_HOST_KUBECONFIG}" kubectl -n "${sa_ns}" create serviceaccount "${sa_name}" \
    --dry-run=client -o yaml | KUBECONFIG="${SPOKE_HOST_KUBECONFIG}" kubectl apply -f - >/dev/null
  KUBECONFIG="${SPOKE_HOST_KUBECONFIG}" kubectl create clusterrolebinding "${sa_name}" \
    --clusterrole=cluster-admin \
    --serviceaccount="${sa_ns}:${sa_name}" \
    --dry-run=client -o yaml | KUBECONFIG="${SPOKE_HOST_KUBECONFIG}" kubectl apply -f - >/dev/null
  ( set +x
    KUBECONFIG="${SPOKE_HOST_KUBECONFIG}" kubectl -n "${sa_ns}" create token "${sa_name}" --duration=24h > "${out_file}"
  )
}

function apply_cluster_profile() {
  local cp_namespace="${1:-default}"
  echo ">> Applying ClusterProfile CR for spoke in namespace ${cp_namespace}"
  kubectl create namespace "${cp_namespace}" --dry-run=client -o yaml | kubectl apply -f - || return 1

  local spoke_endpoint spoke_ca_b64
  spoke_endpoint="$(KUBECONFIG="${SPOKE_KUBECONFIG}" kubectl config view --minify -o jsonpath='{.clusters[0].cluster.server}')" || return 1
  spoke_ca_b64="$(KUBECONFIG="${SPOKE_KUBECONFIG}" kubectl config view --minify --raw -o jsonpath='{.clusters[0].cluster.certificate-authority-data}')" || return 1

  envsubst '${SPOKE_CLUSTER_NAME}' < test/config/multicluster/clusterprofile.yaml.tmpl \
    | kubectl -n "${cp_namespace}" apply -f - || return 1

  # Patch the status subresource separately (kubectl apply drops status).
  local status_tmpdir status_file
  status_tmpdir="$(mktemp -d)" || return 1
  local _status_trap="rm -rf ${status_tmpdir}"
  add_trap "${_status_trap}" EXIT
  status_file="${status_tmpdir}/clusterprofile-status.yaml"

  SPOKE_INTERNAL_ENDPOINT="${spoke_endpoint}" \
  SPOKE_CA_DATA_B64="${spoke_ca_b64}" \
  MC_PROVIDER_NAME="${MC_PROVIDER_NAME}" \
  TRANSITION="$(date -u +%FT%TZ)" \
    envsubst '${SPOKE_INTERNAL_ENDPOINT} ${SPOKE_CA_DATA_B64} ${MC_PROVIDER_NAME} ${TRANSITION}' \
      < test/config/multicluster/clusterprofile-status.yaml.tmpl \
      > "${status_file}" || return 1

  kubectl -n "${cp_namespace}" patch clusterprofile "${SPOKE_CLUSTER_NAME}" \
    --subresource=status --type=merge --patch-file="${status_file}" || return 1

  kubectl -n "${cp_namespace}" wait --for=condition=ControlPlaneHealthy --timeout=30s \
    "clusterprofile/${SPOKE_CLUSTER_NAME}" || return 1
}

function install_access_provider_config() {
  echo ">> Building token-exec-plugin image via ko"
  local plugin_image
  plugin_image="$(ko build ./test/cmd/token-exec-plugin)" || return 1
  if [[ -z "${plugin_image}" ]]; then
    echo "ERROR: ko build did not emit an image reference for token-exec-plugin" >&2
    return 1
  fi
  echo ">> token-exec-plugin image: ${plugin_image}"

  echo ">> Installing access provider ConfigMap/Secret and patching operator deployment"
  local tmpdir
  tmpdir="$(mktemp -d)" || return 1
  add_trap "rm -rf ${tmpdir}" EXIT
  local token_file="${tmpdir}/token"
  _spoke_bootstrap_token "${token_file}" || return 1

  local plugin_command="${MC_PROVIDER_PLUGIN_MOUNT_PATH}/ko-app/token-exec-plugin"
  cat > "${tmpdir}/provider-config.json" <<EOF
{
  "providers": [
    {
      "name": "${MC_PROVIDER_NAME}",
      "execConfig": {
        "apiVersion": "client.authentication.k8s.io/v1",
        "command": "${plugin_command}",
        "args": ["${MC_PROVIDER_TOKEN_MOUNT_PATH}/token"],
        "interactiveMode": "Never"
      }
    }
  ]
}
EOF

  kubectl -n "${TEST_OPERATOR_NAMESPACE}" create configmap "${MC_PROVIDER_CONFIGMAP}" \
    --from-file=config.json="${tmpdir}/provider-config.json" \
    --dry-run=client -o yaml | kubectl apply -f - || return 1

  kubectl -n "${TEST_OPERATOR_NAMESPACE}" create secret generic "${MC_PROVIDER_TOKEN_SECRET}" \
    --from-file=token="${token_file}" \
    --dry-run=client -o yaml | kubectl apply -f - || return 1

  # 0444
  kubectl -n "${TEST_OPERATOR_NAMESPACE}" patch deployment knative-operator \
    --type=json \
    -p "$(cat <<EOF
[
  {"op": "add", "path": "/spec/template/spec/containers/0/args", "value": ["--clusterprofile-provider-file=${MC_PROVIDER_MOUNT_PATH}/config.json"]},
  {"op": "add", "path": "/spec/template/spec/volumes", "value": [
    {"name": "access-config", "configMap": {"name": "${MC_PROVIDER_CONFIGMAP}"}},
    {"name": "provider-token", "secret": {"secretName": "${MC_PROVIDER_TOKEN_SECRET}", "defaultMode": 292}},
    {"name": "access-plugin", "image": {"reference": "${plugin_image}", "pullPolicy": "IfNotPresent"}}
  ]},
  {"op": "add", "path": "/spec/template/spec/containers/0/volumeMounts", "value": [
    {"name": "access-config", "mountPath": "${MC_PROVIDER_MOUNT_PATH}", "readOnly": true},
    {"name": "provider-token", "mountPath": "${MC_PROVIDER_TOKEN_MOUNT_PATH}", "readOnly": true},
    {"name": "access-plugin", "mountPath": "${MC_PROVIDER_PLUGIN_MOUNT_PATH}", "readOnly": true}
  ]}
]
EOF
)" || return 1

  kubectl -n "${TEST_OPERATOR_NAMESPACE}" rollout status deployment/knative-operator --timeout=180s || return 1
}

function setup_multicluster_e2e() {
  local cmd
  for cmd in kind envsubst kubectl ko docker; do
    if ! command -v "${cmd}" >/dev/null 2>&1; then
      echo "ERROR: required command not found: ${cmd}" >&2
      return 1
    fi
  done

  create_spoke_cluster || return 1
  install_cluster_inventory_crd || return 1
  install_access_provider_config || return 1
  apply_cluster_profile "default" || return 1
}
