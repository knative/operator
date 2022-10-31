#!/usr/bin/env bash

# Copyright 2022 The Knative Authors
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

VERSION=1.6.0

rm -rf bundle
kustomize build config/manifests | operator-sdk generate bundle -q --overwrite --version $VERSION
operator-sdk bundle validate ./bundle

# Rename the files of the manifests to conform the naming convention
array=(config-logging_v1_configmap.yaml config-observability_v1_configmap.yaml operator-webhook-certs_v1_secret.yaml operator-webhook_v1_service.yaml)
for file in "${array[@]}"
do
	mv bundle/manifests/${file} bundle/manifests/"knative-operator-"${file}
done

mv bundle/manifests/operator.knative.dev_knativeeventings.yaml bundle/manifests/knativeeventings.operator.knative.dev.crd.yaml
mv bundle/manifests/operator.knative.dev_knativeservings.yaml bundle/manifests/knativeservings.operator.knative.dev.crd.yaml
mv bundle/manifests/knative-operator.clusterserviceversion.yaml bundle/manifests/knative-operator.v${VERSION}.clusterserviceversion.yaml

# Replace the labels with the version number
find bundle/manifests -type f -name "*.yaml" -print0 | xargs -0 sed -i.bak "s/: devel/: "v${VERSION}"/"

# Replace the namespace for the webhooks in the CRDs.
# Openratorhub.io leverages operators as the namespace for the operator.
readonly NS_REPLACE_FILES=(knativeeventings.operator.knative.dev.crd.yaml knativeservings.operator.knative.dev.crd.yaml
 knative-serving-operator-aggregated-stable_rbac.authorization.k8s.io_v1_clusterrolebinding.yaml knative-serving-operator-aggregated_rbac.authorization.k8s.io_v1_clusterrolebinding.yaml
 knative-eventing-operator-aggregated-stable_rbac.authorization.k8s.io_v1_clusterrolebinding.yaml knative-eventing-operator-aggregated_rbac.authorization.k8s.io_v1_clusterrolebinding.yaml)
for FILE in ${NS_REPLACE_FILES[@]}; do
  sed -i.bak "s/namespace: default/namespace: operators/" bundle/manifests/${FILE}
done

# Replace the images
OPERATOR_IMAGE="ko://knative.dev/operator/cmd/operator"
RE_OPERATOR_IMAGE="gcr.io/knative-releases/knative.dev/operator/cmd/operator:v${VERSION}"
WEBHOOK_IMAGE="ko://knative.dev/operator/cmd/webhook"
RE_WEBHOOK_IMAGE="gcr.io/knative-releases/knative.dev/operator/cmd/webhook:v${VERSION}"
FAKE_OPERATOR_NAME="knative-operator-fake"
OPERATOR_NAME="knative-operator"
readonly AGGREGATION_RULES_FILES=(knative-serving-operator-aggregated-stable_rbac.authorization.k8s.io_v1_clusterrolebinding.yaml
 knative-serving-operator-aggregated_rbac.authorization.k8s.io_v1_clusterrolebinding.yaml knative-eventing-operator-aggregated-stable_rbac.authorization.k8s.io_v1_clusterrolebinding.yaml
 knative-eventing-operator-aggregated_rbac.authorization.k8s.io_v1_clusterrolebinding.yaml)

sed -i.bak 's|'${OPERATOR_IMAGE}'|'${RE_OPERATOR_IMAGE}'|' bundle/manifests/knative-operator.v${VERSION}.clusterserviceversion.yaml
sed -i.bak 's|'${WEBHOOK_IMAGE}'|'${RE_WEBHOOK_IMAGE}'|' bundle/manifests/knative-operator.v${VERSION}.clusterserviceversion.yaml

for FILE in ${AGGREGATION_RULES_FILES[@]}; do
  sed -i.bak 's|'${FAKE_OPERATOR_NAME}'|'${OPERATOR_NAME}'|' bundle/manifests/${FILE}
done

# As Knative Operator does no leverage the operator-sdk to generate the operator's code boilerplate,
# we need to remove the webhookdefinitions stanza in the CSV file. The deployment operator-webhook is
# treated as separate deployment.
WEBHOOK_DEFINITIONS="webhookdefinitions"
CSV_FILE="bundle/manifests/knative-operator.v${VERSION}.clusterserviceversion.yaml"

if grep -q ${WEBHOOK_DEFINITIONS} ${CSV_FILE}
then
    LINE_NUM=`cat -n ${CSV_FILE} | grep ${WEBHOOK_DEFINITIONS} | awk '{print $1}'`
    sed -i.bak -n "1,$(( LINE_NUM - 1 )) p; $LINE_NUM q" ${CSV_FILE}
fi

# Remove the file for the fake service account
rm bundle/manifests/knative-operator-fake_v1_serviceaccount.yaml

# Remove all redundant files
rm -rf bundle/manifests/*.bak
