
# Image URL to use all building/pushing image targets
IMG ?= openyurt/yurt-device-controller:latest
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true,preserveUnknownFields=false"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: manager

# Run tests
ENVTEST_ASSETS_DIR=$(shell pwd)/testbin
test: generate fmt vet manifests
	mkdir -p ${ENVTEST_ASSETS_DIR}
	test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.7.0/hack/setup-envtest.sh
	source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test ./... -coverprofile cover.out

# Build binaries
# ARGS:
#   GOOS: the target operating system (i.e., linux, darwin)
#   GOARCH: the target architecture (i.e., amd64, arm64, arm)
#
# Examples:
#   # compile yurt-device-controller with architectures arm64,
#   make build GOOS=linux GOARCH=arm64
#
#   # build binary in the host environment
#   make build
build:
	bash hack/make-rules/build.sh

# Build binaries and docker images.
# NOTE: this rule can take time, as we build binaries inside containers
#
# ARGS:
#   REPO: image repo.
#   TAG:  image tag
#   ARCH: list of target architectures.
#   REGION: in which region this rule is executed, if in mainland China,
#   	set it as cn.
#
# Examples:
#   # compile yurt-device-controller with architectures arm64 and amd64 in the mainland China,
#   # generate images named openyurt/yurt-device-controller:latest-arm64 and openyurt/yurt-device-controller:latest-amd64
#   make release REPO="openyurt" TAG="latest" ARCH="arm64 amd64" REGION=cn
#
#   # compile yurt-device-controller with all architectures (i.e., amd64, arm64, arm)
#   make release
release:
	bash hack/make-rules/release-images.sh

clean:
	-rm -Rf bin
	-rm -Rf _output
	-rm -Rf dockerbuild

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests
	go run ./main.go

# Install CRDs into a cluster
install: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

# Uninstall CRDs from a cluster
uninstall: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests kustomize
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

gen-yaml: manifests kustomize
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default

# UnDeploy controller from the configured Kubernetes cluster in ~/.kube/config
undeploy:
	$(KUSTOMIZE) build config/default | kubectl delete -f -

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Generate code
generate: controller-gen
	go generate ./...
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" \
		paths="./api/v1alpha1/device_types.go" \
		paths="./api/v1alpha1/deviceservice_types.go" \
		paths="./api/v1alpha1/deviceprofile_types.go" \
		paths="./api/v1alpha1/groupversion_info.go"

# Download controller-gen locally if necessary
CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
controller-gen:
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.4.1)

# Download kustomize locally if necessary
KUSTOMIZE = $(shell pwd)/bin/kustomize
kustomize:
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v3@v3.8.7)

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go get $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef
