#!/usr/bin/env bash

# Copyright 2021 The OpenYurt Authors.
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

set -x

YURT_IMAGE_DIR=${YURT_OUTPUT_DIR}/images
DOCKER_BUILD_BASE_IDR=$YURT_ROOT/dockerbuild
YURT_BUILD_IMAGE="golang:1.15-alpine"

readonly -a SUPPORTED_ARCH=(
    amd64
    arm64
    arm
)

readonly SUPPORTED_OS=linux
readonly bin_target=${BIN_NAME:-yurt-device-controller}
readonly -a target_arch=(${ARCH[@]:-${SUPPORTED_ARCH[@]}})
readonly region=${REGION:-us}
readonly image_base_name="${REPO}/${bin_target}:${TAG}"

build_multi_arch_binaries() {
    local docker_yurt_root="/opt/src"
    local docker_run_opts=(
        "-i"
        "--rm"
        "--network host"
        "-v ${YURT_ROOT}:${docker_yurt_root}"
        "--env CGO_ENABLED=0"
        "--env GOOS=${SUPPORTED_OS}"
        "--env PROJECT_PREFIX=${PROJECT_PREFIX}"
        "--env LABEL_PREFIX=${LABEL_PREFIX}"
        "--env GIT_VERSION=${GIT_VERSION}"
        "--env GIT_COMMIT=${GIT_COMMIT}"
        "--env BUILD_DATE=${BUILD_DATE}"
    )
    # use goproxy if build from inside mainland China
    [[ $region == "cn" ]] && docker_run_opts+=("--env GOPROXY=https://goproxy.cn")

    # use proxy if set
    [[ -n ${http_proxy+x} ]] && docker_run_opts+=("--env http_proxy=${http_proxy}")
    [[ -n ${https_proxy+x} ]] && docker_run_opts+=("--env https_proxy=${https_proxy}")

    local docker_run_cmd=(
        "/bin/sh"
        "-xe"
        "-c"
    )

    local sub_commands="sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories; \
        apk --no-cache add bash git; \
        cd ${docker_yurt_root}; umask 0022; \
        rm -rf ${YURT_BIN_DIR}/* ; \
	git config --global --add safe.directory ${docker_yurt_root};"

    for arch in ${target_arch[@]}; do
      sub_commands+="GOARCH=$arch bash ./hack/make-rules/build.sh ${bin_target}; "
    done
    sub_commands+="chown -R $(id -u):$(id -g) ${docker_yurt_root}/_output"

    docker run ${docker_run_opts[@]} ${YURT_BUILD_IMAGE} ${docker_run_cmd[@]} "${sub_commands}"
}

build_docker_image() {
    for arch in ${target_arch[@]}; do
       local binary_name=${bin_target}
       local binary_path=${YURT_BIN_DIR}/${SUPPORTED_OS}/${arch}/${binary_name}

       if [ -f ${binary_path} ]; then
           local docker_build_path=${DOCKER_BUILD_BASE_IDR}/${SUPPORTED_OS}/${arch}
           local docker_file_path=${docker_build_path}/Dockerfile.${binary_name}-${arch}
           mkdir -p ${docker_build_path}

           local base_image="gcr.io/distroless/static:nonroot-${arch}"
           cat <<EOF > "${docker_file_path}"
FROM ${base_image}
WORKDIR /
COPY ${binary_name} .
USER 65532:65532

ENTRYPOINT ["/${binary_name}"]
EOF

           local yurt_component_image_name="${image_base_name}-${arch}"
           ln "${binary_path}" "${docker_build_path}/${binary_name}"
           docker build --no-cache -t "${yurt_component_image_name}" -f "${docker_file_path}" ${docker_build_path}
           docker save ${yurt_component_image_name} > ${YURT_IMAGE_DIR}/${binary_name}-${SUPPORTED_OS}-${arch}.tar
           rm -rf ${docker_build_path}
       fi
    done
}

build_images() {
    # Always clean up before generating the image
    rm -Rf ${YURT_OUTPUT_DIR}
    rm -Rf ${DOCKER_BUILD_BASE_IDR}
    mkdir -p ${YURT_BIN_DIR}
    mkdir -p ${YURT_IMAGE_DIR}
    mkdir -p ${DOCKER_BUILD_BASE_IDR}
    
    build_multi_arch_binaries
    build_docker_image
}
