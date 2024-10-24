#!/bin/bash

# Copyright 2025 The Knative Authors
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

set -e

# GitHub Repository and API token
REPO="knative/operator"
CHART_NAME="knative-operator"

# GitHub API URL for releases
GITHUB_API="https://api.github.com/repos/$REPO/releases"

# Create an empty index.yaml file with 2-space indentation
cat > index.yaml <<EOF
apiVersion: v1
entries:
  $CHART_NAME:
EOF

# Function to fetch all releases and filter .tgz files
fetch_tgz_assets() {
  echo "Fetching release assets from GitHub API..." >&2
  
  # Get response from GitHub API
  response=$(curl "$GITHUB_API")
  
  # Check if the response is valid JSON
  if ! echo "$response" | jq 1>/dev/null 2>&1; then
    echo "(fetch_tgz_assets) Error: The response is not valid JSON. Here's the raw response:" >&2
    echo "$response" >&2
    exit 1
  fi

  # Parse the response using jq to get the list of .tgz files
  echo "$response" | jq -c '.[] | .assets[] | select(.name | test("'$CHART_NAME'-(v?\\d+\\.\\d+\\.\\d+)\\.tgz")) | {url: .browser_download_url, name: .name, published: .updated_at}'
}

# Function to process each .tgz file and append chart metadata to index.yaml
process_tgz() {
  local url=$1
  local name=$2
  local published=$3

  echo "Processing $name from $url" >&2

  # Download the .tgz file
  curl -L -s -o "$name" "$url"

  # Extract the Chart.yaml and values.yaml
  tar -xf "$name" "$CHART_NAME/Chart.yaml" "$CHART_NAME/values.yaml"

  # Parse description from Chart.yaml
  DESCRIPTION=$(yq -r '.description' $CHART_NAME/Chart.yaml)

  # Parse version from Chart.yaml (used as appVersion)
  CHART_VERSION="$(yq -r '.version' $CHART_NAME/Chart.yaml)"

  # Calculate the SHA-256 digest
  DIGEST=$(sha256sum "$name" | cut -d' ' -f1)

  # Append the chart metadata under the existing $CHART_NAME key
  cat >> index.yaml <<EOF
  - name: "$CHART_NAME"
    apiVersion: v2
    version: "v$CHART_VERSION"
    appVersion: "$CHART_VERSION"
    description: "$DESCRIPTION"
    created: "$published"
    urls:
    - "$url"
    digest: "$DIGEST"
EOF

  # Cleanup
  rm -f "$name"
  rm -f $CHART_NAME/Chart.yaml $CHART_NAME/values.yaml
}

# Fetch all .tgz assets
tgz_assets=$(fetch_tgz_assets)

# Loop through all the assets and process them
echo "$tgz_assets" | while read -r asset; do
  # Check if each asset is valid JSON
  if ! echo "$asset" | jq '.' > /dev/null 2>&1; then
    echo "Error: Invalid JSON in asset line. Here's the raw asset line:" >&2
    echo "$asset" >&2
    continue
  fi

  # Parse fields from the asset JSON
  url=$(echo "$asset" | jq -r '.url')
  name=$(echo "$asset" | jq -r '.name' | sed 's/.tgz$//')  # Strip ".tgz" from name
  published=$(echo "$asset" | jq -r '.published')

  process_tgz "$url" "$name" "$published"
done

echo "index.yaml generated successfully!"
