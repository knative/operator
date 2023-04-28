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

# The previous serving release, installed by the operator. This value should be in the semantic format of major.minor.
readonly PREVIOUS_SERVING_RELEASE_VERSION="1.10"
export PREVIOUS_SERVING_RELEASE_VERSION
# The previous eventing release, installed by the operator. This value should be in the semantic format of major.minor.
readonly PREVIOUS_EVENTING_RELEASE_VERSION="1.10"
export PREVIOUS_EVENTING_RELEASE_VERSION
# The target serving/eventing release to upgrade, installed by the operator. It can be a release available under
# kodata or an incoming new release. This value should be in the semantic format of major.minor.
readonly TARGET_RELEASE_VERSION="latest"
export TARGET_RELEASE_VERSION
# This is the branch name of knative repos, where we run the upgrade tests.
readonly KNATIVE_REPO_BRANCH="${PULL_BASE_REF}"
export KNATIVE_REPO_BRANCH
