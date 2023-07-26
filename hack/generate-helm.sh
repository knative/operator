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

VERSION=1.10.1

# The directory used to save the helm templates.
readonly TARGET_DIR="charts/knative-operator"

# Create the directory, if it does not exist.
mkdir -p ${TARGET_DIR}/templates

# Generate the template based on the yaml files under config
echo "" > ${TARGET_DIR}/templates/operator.yaml
for filename in config/*.yaml; do
  cat $filename >> ${TARGET_DIR}/templates/operator.yaml
  echo -e "\n---" >> ${TARGET_DIR}/templates/operator.yaml
done

# Replace the namespace and images with the helm parameters
sed -i.bak 's/namespace: default/namespace: "{{ .Release.Namespace }}"/g' ${TARGET_DIR}/templates/operator.yaml
sed -i.bak 's/image: ko:\/\/knative.dev\/operator\/cmd\/operator/image: "{{ .Values.knative_operator.knative_operator.image }}:{{ .Values.knative_operator.knative_operator.tag }}"/g' ${TARGET_DIR}/templates/operator.yaml
sed -i.bak 's/image: ko:\/\/knative.dev\/operator\/cmd\/webhook/image: "{{ .Values.knative_operator.operator_webhook.image }}:{{ .Values.knative_operator.operator_webhook.tag }}"/g' ${TARGET_DIR}/templates/operator.yaml
sed -i.bak 's/operator.knative.dev\/release: devel/operator.knative.dev\/release: "v{{ .Chart.Version }}"/g' ${TARGET_DIR}/templates/operator.yaml
sed -i.bak 's/app.kubernetes.io\/version: devel/app.kubernetes.io\/version: "{{ .Chart.Version }}"/g' ${TARGET_DIR}/templates/operator.yaml

rm ${TARGET_DIR}/templates/operator.yaml.bak