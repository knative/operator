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

# Set the version and tag in Chart.yaml and values.yaml
VERSION=v1.15.2
if [[ -n "${TAG:-}" ]]; then
  VERSION=${TAG}
fi

# Copy the base file and directories into the directory charts
rm -rf charts
cp -R config/charts charts

# The directory used to save the helm templates.
readonly CHARTS_DIR="charts"
readonly NAME="knative-operator"
readonly TARGET_DIR="${CHARTS_DIR}/${NAME}"

# Create the directory, if it does not exist.
mkdir -p ${TARGET_DIR}/templates

# Generate the template based on the yaml files under config
echo "" > ${TARGET_DIR}/templates/operator.yaml
for filename in config/*.yaml; do
  if [[ $filename == *namespace.yaml ]]; then
      continue
  fi
  cat $filename >> ${TARGET_DIR}/templates/operator.yaml
  echo -e "\n---" >> ${TARGET_DIR}/templates/operator.yaml
done

# Replace the namespace and images with the helm parameters
sed -i.bak 's/namespace: knative-operator/namespace: "{{ .Release.Namespace }}"/g' ${TARGET_DIR}/templates/operator.yaml
sed -i.bak 's/image: ko:\/\/knative.dev\/operator\/cmd\/operator/image: "{{ .Values.knative_operator.knative_operator.image }}:{{ .Values.knative_operator.knative_operator.tag }}"/g' ${TARGET_DIR}/templates/operator.yaml
sed -i.bak 's/image: ko:\/\/knative.dev\/operator\/cmd\/webhook/image: "{{ .Values.knative_operator.operator_webhook.image }}:{{ .Values.knative_operator.operator_webhook.tag }}"/g' ${TARGET_DIR}/templates/operator.yaml
sed -i.bak 's/operator.knative.dev\/release: devel/operator.knative.dev\/release: "v{{ .Chart.Version }}"/g' ${TARGET_DIR}/templates/operator.yaml
sed -i.bak 's/app.kubernetes.io\/version: devel/app.kubernetes.io\/version: "{{ .Chart.Version }}"/g' ${TARGET_DIR}/templates/operator.yaml
sed -i.bak 's/value: ""/value: "{{ .Values.knative_operator.kubernetes_min_version }}"/g' ${TARGET_DIR}/templates/operator.yaml

rm ${TARGET_DIR}/templates/operator.yaml.bak

sed -i.bak "s/{{ version }}/${VERSION:1}/g" ${TARGET_DIR}/Chart.yaml
sed -i.bak "s/{{ tag }}/${VERSION}/g" ${TARGET_DIR}/values.yaml

rm ${TARGET_DIR}/Chart.yaml.bak
rm ${TARGET_DIR}/values.yaml.bak

cd ${CHARTS_DIR}
helm package knative-operator
