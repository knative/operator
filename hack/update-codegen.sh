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

set -o errexit
set -o nounset
set -o pipefail

source $(dirname $0)/../vendor/knative.dev/hack/codegen-library.sh
source "${CODEGEN_PKG}/kube_codegen.sh"

# If we run with -mod=vendor here, then generate-groups.sh looks for vendor files in the wrong place.
export GOFLAGS=-mod=
readonly RELEASE_VERSION="v1.18"

boilerplate="${REPO_ROOT_DIR}/hack/boilerplate/boilerplate.go.txt"

# download all the configurations for different release versions
group "Downloading releases"
(cd ${REPO_ROOT_DIR}; go run ./cmd/fetcher --release ${RELEASE_VERSION} "$@")

group "Kubernetes Codegen"

kube::codegen::gen_client \
  --boilerplate "${REPO_ROOT_DIR}/hack/boilerplate/boilerplate.go.txt" \
  --output-dir "${REPO_ROOT_DIR}/pkg/client" \
  --output-pkg "knative.dev/operator/pkg/client" \
  --with-watch \
  "${REPO_ROOT_DIR}/pkg/apis"

group "Knative Codegen"

${KNATIVE_CODEGEN_PKG}/hack/generate-knative.sh "injection" \
  knative.dev/operator/pkg/client knative.dev/operator/pkg/apis \
  "operator:v1beta1" \
  --go-header-file "${boilerplate}"

group "Deepcopy Gen"

kube::codegen::gen_helpers \
  --boilerplate "${REPO_ROOT_DIR}/hack/boilerplate/boilerplate.go.txt" \
  "${REPO_ROOT_DIR}/pkg/apis"

group "Update deps post-codegen"

# Make sure our dependencies are up-to-date
${REPO_ROOT_DIR}/hack/update-deps.sh
