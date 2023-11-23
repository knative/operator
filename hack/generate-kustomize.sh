#!/usr/bin/env bash

# Copyright 2023 The Knative Authors
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

rm -rf new_config
mkdir new_config

helm template knative-operator config/charts/knative-operator \
    --set knative_operator.kubernetes_min_version="" \
    >knative-operator-default.yaml

awk '
/# Source:/{
    file=gensub(/.*templates\/(.*)/, "new_config/\\1", "g")
    dir=gensub(/(.*)\/.*\.yaml/, "\\1", "g", file)
    system("mkdir -p " dir "&& touch " file)
    print("---") >file
}

file != "" && !/^--/ && !/# Source:/{
    if ($0 ~ /app.kubernetes.io\/version:/)
        print(gensub(/(.*):.*/, "\\1: devel", "g", $0)) >file
    else if ($0 ~ /gcr.io\/knative-releases\/knative.dev/)
        print(gensub(/(.*image:).*\/(.*):.*/, "\\1 ko://knative.dev/operator/cmd/\\2", "g", $0)) >file
    else
        print >file
}
' knative-operator-default.yaml

cp -r new_config/* config
rm -rf new_config

rm -f knative-operator-default.yaml
