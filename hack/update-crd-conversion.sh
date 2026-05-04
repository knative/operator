#!/usr/bin/env bash

# Copyright 2026 The Knative Authors
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

# Adds the conversion webhook stanza that controller-gen does not emit for the
# operator's single-version CRDs. The release operator.yaml consumes these bases
# directly, so this must run before any chart or release manifest generation.

set -o errexit
set -o nounset
set -o pipefail

REPO_ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CRD_BASES_DIR="${REPO_ROOT_DIR}/config/crd/bases"

echo "Updating CRD conversion webhooks in ${CRD_BASES_DIR}"

for crd_file in \
  "${CRD_BASES_DIR}/operator.knative.dev_knativeeventings.yaml" \
  "${CRD_BASES_DIR}/operator.knative.dev_knativeservings.yaml"; do
  if [[ ! -f "${crd_file}" ]]; then
    echo "ERROR: CRD file not found: ${crd_file}" >&2
    exit 1
  fi

  echo "  Updating $(basename "${crd_file}")"
  tmp="$(mktemp "${crd_file}.XXXXXX")"
  if ! awk '
    function emit_conversion() {
      print "  conversion:"
      print "    strategy: Webhook"
      print "    webhook:"
      print "      conversionReviewVersions: [\"v1beta1\"]"
      print "      clientConfig:"
      print "        service:"
      print "          name: operator-webhook"
      print "          namespace: knative-operator"
      print "          path: /resource-conversion"
    }

    $0 == "  conversion:" {
      emit_conversion()
      inserted = 1
      skipping = 1
      next
    }

    skipping && $0 == "  versions:" {
      skipping = 0
      print
      next
    }

    skipping {
      next
    }

    $0 == "  versions:" && !inserted {
      emit_conversion()
      inserted = 1
      print
      next
    }

    {
      print
    }

    END {
      if (!inserted) {
        print "ERROR: could not find CRD spec.versions anchor" > "/dev/stderr"
        exit 1
      }
    }
  ' "${crd_file}" > "${tmp}"; then
    rm -f "${tmp}"
    exit 1
  fi
  mv "${tmp}" "${crd_file}"
done

echo "Done. CRD conversion webhooks updated."
