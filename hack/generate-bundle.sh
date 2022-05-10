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

VERSION=1.5.0

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
sed -i.bak "s/namespace: default/namespace: operators/" bundle/manifests/knativeeventings.operator.knative.dev.crd.yaml
sed -i.bak "s/namespace: default/namespace: operators/" bundle/manifests/knativeservings.operator.knative.dev.crd.yaml
sed -i.bak "s/namespace: default/namespace: operators/" bundle/manifests/knative-operator-post-install-job-role-binding_rbac.authorization.k8s.io_v1_clusterrolebinding.yaml

# Replace the images
OPERATOR_IMAGE="ko://knative.dev/operator/cmd/operator"
RE_OPERATOR_IMAGE="gcr.io/knative-releases/knative.dev/operator/cmd/operator:v${VERSION}"
WEBHOOK_IMAGE="ko://knative.dev/operator/cmd/webhook"
RE_WEBHOOK_IMAGE="gcr.io/knative-releases/knative.dev/operator/cmd/webhook:v${VERSION}"

sed -i.bak 's|'${OPERATOR_IMAGE}'|'${RE_OPERATOR_IMAGE}'|' bundle/manifests/knative-operator.v${VERSION}.clusterserviceversion.yaml
sed -i.bak 's|'${WEBHOOK_IMAGE}'|'${RE_WEBHOOK_IMAGE}'|' bundle/manifests/knative-operator.v${VERSION}.clusterserviceversion.yaml

rm -rf bundle/manifests/*.bak
