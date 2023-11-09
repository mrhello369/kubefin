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

# For all commands, the working directory is the parent directory(repo root).
REPO_ROOT=$(git rev-parse --show-toplevel)
cd "${REPO_ROOT}"

export GOPATH=$(go env GOPATH | awk -F ':' '{print $1}')
export PATH=$PATH:$GOPATH/bin

boilerplate="${REPO_ROOT}"/hack/boilerplate/boilerplate.go.txt

go_path="${REPO_ROOT}/_go"
cleanup() {
  rm -rf "${go_path}"
}
trap "cleanup" EXIT SIGINT

cleanup

source "${REPO_ROOT}"/hack/utils.sh
create_gopath_tree "${REPO_ROOT}" "${go_path}"
export GOPATH="${go_path}"

echo "Generating with deepcopy-gen"
deepcopy-gen \
  --output-file-base zz_generated.deepcopy \
  --go-header-file "${boilerplate}" \
  --input-dirs github.com/kubefin/kubefin/pkg/apis/insight/v1alpha1

echo "Generating with register-gen"
register-gen \
  --output-file-base zz_generated.register \
  --go-header-file "${boilerplate}" \
  --input-dirs github.com/kubefin/kubefin/pkg/apis/insight/v1alpha1

echo "Generating with conversion-gen"
conversion-gen \
  -O zz_generated.conversion \
  --go-header-file "${boilerplate}" \
  --input-dirs github.com/kubefin/kubefin/pkg/apis/insight/v1alpha1

echo "Generating with client-gen"
client-gen \
  --go-header-file "${boilerplate}" \
  --input-base "" \
  --input github.com/kubefin/kubefin/pkg/apis/insight/v1alpha1 \
  --output-package github.com/kubefin/kubefin/pkg/generated/clientset \
  --clientset-name versioned

echo "Generating with lister-gen"
lister-gen \
  --go-header-file "${boilerplate}" \
  --input-dirs github.com/kubefin/kubefin/pkg/apis/insight/v1alpha1 \
  --output-package github.com/kubefin/kubefin/pkg/generated/listers

echo "Generating with informer-gen"
informer-gen \
  --go-header-file "${boilerplate}" \
  --input-dirs github.com/kubefin/kubefin/pkg/apis/insight/v1alpha1 \
  --versioned-clientset-package github.com/kubefin/kubefin/pkg/generated/clientset/versioned \
  --listers-package github.com/kubefin/kubefin/pkg/generated/listers \
  --output-package github.com/kubefin/kubefin/pkg/generated/informers
