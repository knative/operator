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

# If we run with -mod=vendor here, then generate-groups.sh looks for vendor files in the wrong place.
export GOFLAGS=-mod=
readonly RELEASE_VERSION="v1.9"

boilerplate="${REPO_ROOT_DIR}/hack/boilerplate/boilerplate.go.txt"

# download all the configurations for different release versions
group "Downloading releases"
(cd ${REPO_ROOT_DIR}; go run ./cmd/fetcher --release ${RELEASE_VERSION} "$@")

group "Kubernetes Codegen"

# generate the code with:
# --output-base    because this script should also be able to run inside the vendor dir of
#                  k8s.io/kubernetes. The output-base is needed for the generators to output into the vendor dir
#                  instead of the $GOPATH directly. For normal projects this can be dropped.
${CODEGEN_PKG}/generate-groups.sh "deepcopy,client,informer,lister" \
  knative.dev/operator/pkg/client knative.dev/operator/pkg/apis \
  "operator:v1beta1" \
  --go-header-file "${boilerplate}"

group "Knative Codegen"

${KNATIVE_CODEGEN_PKG}/hack/generate-knative.sh "injection" \
  knative.dev/operator/pkg/client knative.dev/operator/pkg/apis \
  "operator:v1beta1" \
  --go-header-file "${boilerplate}"

group "Deepcopy Gen"

# Depends on generate-groups.sh to install bin/deepcopy-gen
${GOPATH}/bin/deepcopy-gen \
  -O zz_generated.deepcopy \
  --go-header-file "${boilerplate}" \
  -i knative.dev/operator/pkg/apis/operator/base \
  -i knative.dev/operator/pkg/apis/operator/v1beta1

group "Update deps post-codegen"

# Make sure our dependencies are up-to-date
${REPO_ROOT_DIR}/hack/update-deps.sh
