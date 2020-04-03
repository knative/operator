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

# This scripts the Operator Lifecycle Manager (OLM) CatalogSource YAML.
# When CatalogSource is fed to Kubernetes, OLM will pick it up and
# it will be possible to use the operator with OLM.

DIR=${DIR:-$(cd $(dirname "$0")/.. && pwd)}
NAME=${NAME:-$(ls $DIR/deploy/olm-catalog)}

x=( $(echo $NAME | tr '-' ' ') )
DISPLAYNAME=${DISPLAYNAME:=${x[*]^}}

indent() {
  INDENT="      "
  ENDASH="    - "
  sed "s/^/$INDENT/" | sed "s/^${INDENT}\($1\)/${ENDASH}\1/"
}

rm -rf $DIR/.crds
mkdir $DIR/.crds
find $DIR/deploy/olm-catalog -name '*.crd.yaml' | sort -n | xargs -I{} cp {} $DIR/.crds/

CRD=$(cat $(ls $DIR/.crds/*) | grep -v -- "---" | indent apiVersion)
CSV=$(cat $(find $DIR/deploy/olm-catalog -name '*version.yaml' | sort -n) | indent apiVersion)
PKG=$(cat $DIR/deploy/olm-catalog/$NAME/*package.yaml | indent packageName)

cat <<EOF | sed 's/^  *$//'
kind: ConfigMap
apiVersion: v1
metadata:
  name: $NAME

data:
  customResourceDefinitions: |-
$CRD
  clusterServiceVersions: |-
$CSV
  packages: |-
$PKG
---
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: $NAME
spec:
  configMap: $NAME
  displayName: $DISPLAYNAME
  publisher: The Knative Authors
  sourceType: internal
EOF
