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

# This script runs the presubmit tests; it is started by prow for each PR.
# For convenience, it can also be executed manually.
# Running the script without parameters, or with the --all-tests
# flag, causes all tests to be executed, in the right order.
# Use the flags --build-tests, --unit-tests and --integration-tests
# to run a specific set of tests.

export GO111MODULE=on
export GOFLAGS=-mod=

source $(dirname $0)/../vendor/knative.dev/hack/presubmit-tests.sh

# We use the default build, unit and integration test runners.

function integration_tests() {
  local options=""
  local failed=0
  e2e_test="test/e2e-tests.sh"
  echo "Running integration test ${e2e_test}"
  if ! ${e2e_test} ${options}; then
    failed=1
  fi
  return ${failed}
}

main $@
