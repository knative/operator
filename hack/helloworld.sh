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

NAME=hello
TARGET=${USER:-world}

# Create a sample Knative Service
cat <<EOF | kubectl apply -f -
apiVersion: serving.knative.dev/v1alpha1
kind: Service
metadata:
  name: $NAME
spec:
  template:
    spec:
      containers:
        - image: gcr.io/knative-samples/helloworld-go
          env:
            - name: TARGET
              value: $TARGET
EOF

# Wait for the Knative Service to be ready
while output=$(kubectl get ksvc $NAME); do
  echo "$output"
  echo $output | grep True >/dev/null && break
  sleep 2
done

# Parse the URL from the knative service
URL=$(kubectl get ksvc $NAME | grep True | awk '{print $2}')

# Fetch it, accounting for possible istio race conditions
until curl -f $URL; do sleep 2; done
