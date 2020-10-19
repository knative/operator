#!/usr/bin/env bash
#
# Copyright 2018 The Knative Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit

# This function is used to build and publish the test images.
# $1 - the first parameter is a string, specifying a path of the root directory of the project to run the ko command.
# $2 - the second parameter is a string, specifying a sub-directory, where the images are built, under the root directory.
# $3 - the third parameter is a string, specifying the tag of the images.
function upload_test_images() {
  echo ">> Publishing test images"
  # Script needs to be executed from the root directory
  # to pickup .ko.yaml
  local root_dir=$1
  if [ ! -n "$root_dir" ] ; then
    root_dir="$( dirname "$0")/.."
  fi
  cd ${root_dir}
  local image_dir=$2
  if [ ! -n "$image_dir" ] ; then
    image_dir="test/test_images"
  fi
  local docker_tag=$3
  local tag_option=""
  if [ -n "${docker_tag}" ]; then
    tag_option="--tags $docker_tag,latest"
  fi

  # ko resolve is being used for the side-effect of publishing images,
  # so the resulting yaml produced is ignored.
  ko resolve ${tag_option} -RBf "${image_dir}" > /dev/null
}

: ${KO_DOCKER_REPO:?"You must set 'KO_DOCKER_REPO', see DEVELOPMENT.md"}

upload_test_images $@
