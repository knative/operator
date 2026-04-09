#!/usr/bin/env bash

# Copyright 2025 The Knative Authors
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

# This script runs the multi-cluster end-to-end tests for Knative
# Operator. It is started by prow for each PR. For convenience, it can
# also be executed manually.

set -o errexit
set -o pipefail

# Must be set before sourcing e2e-common.sh (hack defaults to gke).
export CLOUD_PROVIDER=kind
export KO_DOCKER_REPO=kind.local
export KIND_CLUSTER_NAME=kind

source "$(dirname "$0")/e2e-common.sh"

: "${SPOKE_CLUSTER_NAME:=spoke}"
: "${SPOKE_KUBECONFIG:=/tmp/spoke.kubeconfig}"
: "${SPOKE_HOST_KUBECONFIG:=/tmp/spoke-host.kubeconfig}"
: "${CLUSTER_INVENTORY_CRD_URL:=https://raw.githubusercontent.com/kubernetes-sigs/cluster-inventory-api/v0.1.0/config/crd/bases/multicluster.x-k8s.io_clusterprofiles.yaml}"
: "${MC_PROVIDER_CONFIGMAP:=clusterprofile-provider-file}"
: "${MC_PROVIDER_TOKEN_SECRET:=clusterprofile-provider-token}"
: "${MC_PROVIDER_MOUNT_PATH:=/etc/cluster-inventory}"
: "${MC_PROVIDER_TOKEN_MOUNT_PATH:=/etc/cluster-inventory/access}"
: "${MC_PROVIDER_PLUGIN_MOUNT_PATH:=/etc/cluster-inventory/plugin}"
: "${MC_PROVIDER_NAME:=e2e-static-token}"
export SPOKE_CLUSTER_NAME SPOKE_KUBECONFIG SPOKE_HOST_KUBECONFIG
export CLUSTER_INVENTORY_CRD_URL MC_PROVIDER_CONFIGMAP MC_PROVIDER_TOKEN_SECRET
export MC_PROVIDER_MOUNT_PATH MC_PROVIDER_TOKEN_MOUNT_PATH
export MC_PROVIDER_PLUGIN_MOUNT_PATH MC_PROVIDER_NAME

function knative_setup() {
  create_namespace
  install_operator
}

# Clean up stale clusters from previous runs, but skip on --run-tests
# re-exec to avoid deleting the hub cluster kubetest2 just brought up.
_run_tests_mode=0
for _arg in "$@"; do
  if [[ "${_arg}" == "--run-tests" ]]; then _run_tests_mode=1; break; fi
done
if (( ! _run_tests_mode )) && command -v kind >/dev/null 2>&1; then
  _existing_clusters="$(kind get clusters 2>/dev/null || true)"
  for _c in "${KIND_CLUSTER_NAME}" "${SPOKE_CLUSTER_NAME}"; do
    if printf '%s\n' "${_existing_clusters}" | grep -qx "${_c}"; then
      echo ">> Pre-run cleanup: deleting stale kind cluster ${_c}"
      kind delete cluster --name "${_c}" || true
    fi
  done
  unset _existing_clusters _c
fi
unset _run_tests_mode _arg

initialize "$@" --cluster-name "${KIND_CLUSTER_NAME}"

header "Setting up spoke kind cluster"
function multicluster_cleanup_trap() {
  dump_hub_state || true
  dump_spoke_state || true
  delete_spoke_cluster || true
}
add_trap multicluster_cleanup_trap EXIT
setup_multicluster_e2e || fail_test "failed to set up spoke cluster"

echo ">> Hub+spoke bootstrap complete; starting multicluster tests"

header "Running multi-cluster e2e for Knative Operator"
failed=0

: "${MULTICLUSTER_TEST_RUN_REGEX:=^TestMulticluster}"
echo ">> Run regex: ${MULTICLUSTER_TEST_RUN_REGEX}"
go_test_e2e -timeout=30m -tags='e2e multicluster' -run "${MULTICLUSTER_TEST_RUN_REGEX}" ./test/e2e || failed=1

(( failed )) && fail_test
success
