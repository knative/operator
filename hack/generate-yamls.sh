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

# This script builds all the YAMLs that Knative operator publishes.
# It may be varied between different branches, of what it does, but the
# following usage must be observed:
#
# generate-yamls.sh  <repo-root-dir> <generated-yaml-list>
#     repo-root-dir         the root directory of the repository.
#     generated-yaml-list   an output file that will contain the list of all
#                           YAML files. The first file listed must be our
#                           manifest that contains all images to be tagged.

# Different versions of our scripts should be able to call this script with
# such assumption so that the test/publishing/tagging steps can evolve
# differently than how the YAMLs are built.

# The following environment variables affect the behavior of this script:
# * `$KO_FLAGS` Any extra flags that will be passed to ko.
# * `$YAML_OUTPUT_DIR` Where to put the generated YAML files, otherwise a
#   random temporary directory will be created. **All existing YAML files in
#   this directory will be deleted.**
# * `$KO_DOCKER_REPO` If not set, use ko.local as the registry.

set -o errexit
set -o pipefail
set -o xtrace

readonly YAML_REPO_ROOT=${1:?"First argument must be the repo root dir"}
readonly YAML_LIST_FILE=${2:?"Second argument must be the output file"}

# Set output directory
if [[ -z "${YAML_OUTPUT_DIR:-}" ]]; then
  readonly YAML_OUTPUT_DIR="${YAML_REPO_ROOT}/output"
  mkdir -p "${YAML_OUTPUT_DIR}"
fi
rm -fr ${YAML_OUTPUT_DIR}/*.yaml

# Generated Knative Operator component YAML files
readonly OPERATOR_YAML=${YAML_OUTPUT_DIR}/operator.yaml

if [[ -n "${TAG:-}" ]]; then
  LABEL_YAML_CMD=(sed -e "s|app.kubernetes.io/version: devel|app.kubernetes.io/version: \"${TAG:1}\"|")
else
  LABEL_YAML_CMD=(cat)
fi

# Flags for all ko commands
KO_YAML_FLAGS="-P"
[[ "${KO_DOCKER_REPO}" != gcr.io/* ]] && KO_YAML_FLAGS=""
readonly KO_YAML_FLAGS="${KO_YAML_FLAGS} ${KO_FLAGS}"

: ${KO_DOCKER_REPO:="ko.local"}
export KO_DOCKER_REPO

cd "${YAML_REPO_ROOT}"

echo "Building Knative Operator"
ko resolve ${KO_YAML_FLAGS} -f config/ | "${LABEL_YAML_CMD[@]}" > "${OPERATOR_YAML}"
all_yamls=(${OPERATOR_YAML})

if [ -d "${YAML_REPO_ROOT}/config/post-install" ]; then
  readonly OPERATOR_POST_INSTALL_YAML=${YAML_OUTPUT_DIR}/"operator-post-install.yaml"

  echo "Resolving post install manifests"
  ko resolve ${KO_YAML_FLAGS} -f config/post-install/ | "${LABEL_YAML_CMD[@]}" > "${OPERATOR_POST_INSTALL_YAML}"
  all_yamls+=(${OPERATOR_POST_INSTALL_YAML})
fi

echo "All manifests generated"

./hack/generate-helm.sh
HELM_CHARTS=${YAML_REPO_ROOT}/charts/knative-operator-${TAG:1}.tgz
all_yamls+=(${HELM_CHARTS})

for yaml in "${!all_yamls[@]}"; do
  echo "${all_yamls[${yaml}]}" >> ${YAML_LIST_FILE}
done
