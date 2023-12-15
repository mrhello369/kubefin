#!/usr/bin/env bash

# Copyright 2023 The KubeFin Authors
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

# This script builds go components.
# You can set the platform to build with BUILD_PLATFORMS, with format: `<os>/<arch>`
# And binaries will be put in `_output/<os>/<arch>/`
#
# Usage:
#   hack/build.sh <target>
# Args:
#   $1:              target to build
# Environments:
#   BUILD_PLATFORMS: platforms to build. You can set one or more platforms separated by comma.
#                    e.g.: linux/amd64,linux/arm64
#   LDFLAGS          pass to the `-ldflags` parameter of go build
# Examples:
#   hack/build.sh kubefin-agent
#   BUILD_PLATFORMS=linux/amd64,linux/arm64 hack/build.sh kubefin-agent

REPO_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
source "${REPO_ROOT}/hack/utils.sh"

LDFLAGS="$(version_ldflags) ${LDFLAGS:-}"

function build_binary() {
  local -r target=$1

  IFS="," read -ra platforms <<< "${BUILD_PLATFORMS:-}"
  if [[ ${#platforms[@]} -eq 0 ]]; then
    platforms=("$(host_platform)")
  fi

  for platform in "${platforms[@]}"; do
    echo "!!! Building ${target} for ${platform}:"
    build_binary_for_platform "${target}" "${platform}"
  done
}

function build_binary_for_platform() {
  local -r target=$1
  local -r platform=$2
  local -r os=${platform%/*}
  local -r arch=${platform##*/}

  local gopkg="${KUBEFIN_GO_PACKAGE}/$(get_target_source $target)"
  set -x
  CGO_ENABLED=0 GOOS=${os} GOARCH=${arch} go build \
      -ldflags "${LDFLAGS:-}" \
      -o "_output/bin/${platform}/$target" \
      "${gopkg}"
  set +x
}

build_binary "$@"
