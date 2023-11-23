#!/usr/bin/env bash

# Copyright 2022 The KubeFin Authors
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
set -o pipefail

TAG=${TAG:-"latest"}
readonly REPO_ROOT=${1:?"First argument must be the repo root dir"}
readonly YAML_OUTPUT_DIR=${2:?"Second argument must be the output dir"}

# Cleanup output directory
rm -fr ${YAML_OUTPUT_DIR}/*.yaml

# generate the cluster deployment yaml
hack/init-primary-config.sh auto cluster-0 true "${TAG}"
hack/init-secondary-config.sh auto cluster-1 http://127.0.0.1/api/v1/push

# Generated KubeFin component YAML files
readonly KUBEFIN_CRD_YAML=${YAML_OUTPUT_DIR}/kubefin-crd.yaml
readonly KUBEFIN_PRIMARY_YAML=${YAML_OUTPUT_DIR}/kubefin-primary.yaml
readonly KUBEFIN_SECONDARY_YAML=${YAML_OUTPUT_DIR}/kubefin-secondary.yaml

# Flags for all ko commands
# In order to push image to dockerhub, we use flag '-B' to ignore some parts
# Referring:https://github.com/ko-build/ko/issues/44
KO_YAML_FLAGS="-B"
KO_FLAGS="${KO_FLAGS:-}"
[[ "${KO_DOCKER_REPO}" != docker.io/* ]] && KO_YAML_FLAGS=""

if [[ "${KO_FLAGS}" != *"--platform"* ]]; then
  KO_YAML_FLAGS="${KO_YAML_FLAGS} --platform=linux/amd64"
fi

readonly KO_YAML_FLAGS="${KO_YAML_FLAGS} ${KO_FLAGS}"

if [[ -n "${TAG:-}" ]]; then
  LABEL_YAML_CMD=(sed -e "s|app.kubernetes.io/version: devel|app.kubernetes.io/version: \"${TAG:1}\"|")
else
  LABEL_YAML_CMD=(cat)
fi

: "${KO_DOCKER_REPO:="ko.local"}"
export KO_DOCKER_REPO

cd "${YAML_REPO_ROOT}"

# delete debug component for release
rm -rf config_primary/third_party/grafana.yaml

echo "Building KubeFin"
ko resolve ${KO_YAML_FLAGS} -t ${TAG} --tag-only -B -R -f config_primary/crds | "${LABEL_YAML_CMD[@]}" > "${KUBEFIN_CRD_YAML}"
ko resolve ${KO_YAML_FLAGS} -t ${TAG} --tag-only -B -R -f config_secondary/core | "${LABEL_YAML_CMD[@]}" >> "${KUBEFIN_SECONDARY_YAML}"
ko resolve ${KO_YAML_FLAGS} -t ${TAG} --tag-only -B -R -f config_primary/core,config_primary/third_party | "${LABEL_YAML_CMD[@]}" > "${KUBEFIN_PRIMARY_YAML}"


echo "All manifests are generated"
