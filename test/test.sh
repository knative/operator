#!/usr/bin/env bash

readonly ISTIO_VERSION="1.4-latest"

istio_version="istio-${ISTIO_VERSION}"
if [[ ${istio_version} == *-latest ]] ; then
  istio_version=$(curl https://raw.githubusercontent.com/knative/serving/v0.13.0/third_party/${istio_version})
fi

echo $istio_version

TEST_NAMESPACE="knative-serving";
LATEST_SERVING_RELEASE_VERSION="v0.12.1";
#local pods="$(kubectl get pods --no-headers -n $1 2>/dev/null)"
list_resources="all,cm,crd,sa,ClusterRole,ClusterRoleBinding,Image,ValidatingWebhookConfiguration,\
MutatingWebhookConfiguration,Secret,RoleBinding,APIService,Gateway"
result="$(kubectl get ${list_resources} -l serving.knative.dev/release=${LATEST_SERVING_RELEASE_VERSION} --all-namespaces 2>/dev/null)"

#result="$(kubectl get all --no-headers -n ${TEST_NAMESPACE} -l serving.knative.dev/release!=${LATEST_SERVING_RELEASE_VERSION} 2>/dev/null)"
echo "${result}"
if [[ ! -z ${result} ]] ; then
  echo "not empty."
fi
