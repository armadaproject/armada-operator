# VERSION defines the project version for the bundle.
# Update this value when you upgrade the version of your project.
# To re-generate a bundle for another specific version without changing the standard setup, you can:
# - use the VERSION as arg of the bundle target (e.g make bundle VERSION=0.0.2)
# - use environment variables to overwrite this value (e.g export VERSION=0.0.2)
VERSION ?= 0.0.1

# CHANNELS define the bundle channels used in the bundle.
# Add a new line here if you would like to change its default config. (E.g CHANNELS = "candidate,fast,stable")
# To re-generate a bundle for other specific channels without changing the standard setup, you can:
# - use the CHANNELS as arg of the bundle target (e.g make bundle CHANNELS=candidate,fast,stable)
# - use environment variables to overwrite this value (e.g export CHANNELS="candidate,fast,stable")
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif

# DEFAULT_CHANNEL defines the default channel used in the bundle.
# Add a new line here if you would like to change its default config. (E.g DEFAULT_CHANNEL = "stable")
# To re-generate a bundle for any other default channel without changing the default setup, you can:
# - use the DEFAULT_CHANNEL as arg of the bundle target (e.g make bundle DEFAULT_CHANNEL=stable)
# - use environment variables to overwrite this value (e.g export DEFAULT_CHANNEL="stable")
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# IMAGE_TAG_BASE defines the docker.io namespace and part of the image name for remote images.
# This variable is used to construct full image tags for bundle and catalog images.
#
# For example, running 'make bundle-build bundle-push catalog-build catalog-push' will build and push both
# armadaproject.io/armada-operator-bundle:$VERSION and armadaproject.io/armada-operator-catalog:$VERSION.
IMAGE_TAG_BASE ?= armadaproject.io/armada-operator

# BUNDLE_IMG defines the image:tag used for the bundle.
# You can use it as an arg. (E.g make bundle-build BUNDLE_IMG=<some-registry>/<project-name-bundle>:<tag>)
BUNDLE_IMG ?= $(IMAGE_TAG_BASE)-bundle:v$(VERSION)

# BUNDLE_GEN_FLAGS are the flags passed to the operator-sdk generate bundle command
BUNDLE_GEN_FLAGS ?= -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)

# USE_IMAGE_DIGESTS defines if images are resolved via tags or digests
# You can enable this value if you would like to use SHA Based Digests
# To enable set flag to true
USE_IMAGE_DIGESTS ?= false
ifeq ($(USE_IMAGE_DIGESTS), true)
	BUNDLE_GEN_FLAGS += --use-image-digests
endif

# Image URL to use all building/pushing image targets
IMG ?= armada-operator:latest
# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.24.2

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

ARCH=$(shell go env GOARCH)
KIND_DEV_CLUSTER_NAME=armada-operator-dev-env

.PHONY: all
all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: mock
mock: mockgen
	$(RM) test/k8sclient/mock_client.go
	mockgen -destination=test/k8sclient/mock_client.go -package=k8sclient "github.com/armadaproject/armada-operator/test/k8sclient" Client

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: lint
lint:
	golangci-lint run

.PHONY: lint-fix
lint-fix:
	golangci-lint run --fix

.PHONY: test
test: manifests generate fmt vet gotestsum ## Run tests.
	$(GOTESTSUM) -- ./controllers/... -coverprofile operator.out

.PHONY: test-integration
test-integration: manifests generate fmt vet gotestsum envtest ## Run integration tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" $(GOTESTSUM) -- ./test/... ./apis/...
.PHONY: kind-create
kind-create: kind
	kind create cluster --config hack/kind-config.yaml
.PHONY: test-e2e-cli

.PHONY: test-e2e
test-e2e: envtest go-release-build
	kind create cluster --name armada-operator-e2e-test --config hack/kind-config.yaml
	kind load docker-image --name armada-operator-e2e-test ${IMG}
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test -v ./test/e2e
	kind delete cluster --name armada-operator-e2e-test 

# Integration test without Ginkgo colorized output and control chars, for logging purposes
.PHONY: test-integration-debug
test-integration-debug: manifests generate fmt vet gotestsum envtest ## Run integration tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test -v ./test/... ./apis/... --coverprofile integration.out -args --ginkgo.no-color

##@ Build

.PHONY: build
build: generate fmt vet ## Build manager binary.
	go build -o bin/manager main.go

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	go run ./main.go

# Go Release Build
.PHONY: go-release-build
go-release-build: goreleaser
	$(GORELEASER) release --rm-dist --snapshot
# If you wish built the manager image targeting other platforms you can use the --platform flag.
# (i.e. docker build --platform linux/arm64 ). However, you must enable docker buildKit for it.
# More info: https://docs.docker.com/develop/develop-images/build_enhancements/
.PHONY: docker-build
docker-build: test ## Build docker image with the manager.
	docker build -t ${IMG} .

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	docker push ${IMG}

# Load image to kind
.PHONY: load-image
load-image:
	kind load docker-image --name $(KIND_DEV_CLUSTER_NAME) ${IMG}

# PLATFORMS defines the target platforms for  the manager image be build to provide support to multiple
# architectures. (i.e. make docker-buildx IMG=myregistry/mypoperator:0.0.1). To use this option you need to:
# - able to use docker buildx . More info: https://docs.docker.com/build/buildx/
# - have enable BuildKit, More info: https://docs.docker.com/develop/develop-images/build_enhancements/
# - be able to push the image for your registry (i.e. if you do not inform a valid value via IMG=<myregistry/image:<tag>> than the export will fail)
# To properly provided solutions that supports more than one platform you should use this option.
PLATFORMS ?= linux/arm64,linux/amd64,linux/s390x,linux/ppc64le
.PHONY: docker-buildx
docker-buildx: test ## Build and push docker image for the manager for cross-platform support
	# copy existing Dockerfile and insert --platform=${BUILDPLATFORM} into Dockerfile.cross, and preserve the original Dockerfile
	sed -e '1 s/\(^FROM\)/FROM --platform=\$$\{BUILDPLATFORM\}/; t' -e ' 1,// s//FROM --platform=\$$\{BUILDPLATFORM\}/' Dockerfile > Dockerfile.cross
	- docker buildx create --name project-v3-builder
	docker buildx use project-v3-builder
	- docker buildx build --push --platform=$(PLATFORMS) --tag ${IMG} -f Dockerfile.cross
	- docker buildx rm project-v3-builder
	rm Dockerfile.cross

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy
deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

.PHONY: deploy-to-kind
deploy-to-kind: dev-setup docker-build load-image deploy

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/default | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: generate-helm-chart
generate-helm-chart: manifests kustomize helmify
	$(KUSTOMIZE) build config/default | $(HELMIFY) -crd-dir deployment/armada-operator

## Kubernetes Dependencies
CERT_MANAGER_MANIFEST ?= "https://github.com/cert-manager/cert-manager/releases/download/v1.6.3/cert-manager.yaml"
.PHONY: install-cert-manager
install-cert-manager:
	kubectl apply -f ${CERT_MANAGER_MANIFEST}

.PHONY: uninstall-cert-manager
uninstall-cert-manager:
	kubectl delete -f ${CERT_MANAGER_MANIFEST}

INGRESS_MANIFEST ?= "https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml"
.PHONY: install-ingress-controller
install-ingress-controller:
	kubectl apply -f ${INGRESS_MANIFEST}
.PHONY: uninstall-ingress-controller
uninstall-ingress-controller:
	kubectl delete -f ${INGRESS_MANIFEST}

PULSAR_IMAGE="apachepulsar/pulsar"
ifeq ($(ARCH), arm64)
   PULSAR_IMAGE="richgross/pulsar:2.11.0"
endif
.PHONY: install-pulsar
install-pulsar:
	docker pull $(PULSAR_IMAGE)
	kind load docker-image --name $(KIND_DEV_CLUSTER_NAME) $(PULSAR_IMAGE)
	cat dev/manifests/pulsar.yaml | PULSAR_IMAGE=$(PULSAR_IMAGE) envsubst | kubectl apply -n armada -f -

.PHONY: helm-bitnami
helm-bitnami: helm
	$(HELM) repo add bitnami https://charts.bitnami.com/bitnami
	$(HELM) repo update

.PHONY: helm-install-postgres
helm-install-postgres: helm-bitnami
	docker pull postgres:15.2-alpine
	kind load docker-image --name $(KIND_DEV_CLUSTER_NAME) postgres:15.2-alpine
	$(HELM) install postgresql -n armada -f ./dev/helm-charts/postgres_bitnami_values.yaml bitnami/postgresql

.PHONY: helm-install-redis
helm-install-redis: helm-bitnami
	$(HELM) install redis -n armada -f ./dev/helm-charts/redis_bitnami_values.yaml bitnami/redis

PROMETHEUS_OPERATOR_VERSION=v0.62.0
.PHONY: dev-install-prometheus-operator
dev-install-prometheus-operator:
	curl -sL https://github.com/prometheus-operator/prometheus-operator/releases/download/${PROMETHEUS_OPERATOR_VERSION}/bundle.yaml | sed -e 's/namespace: default/namespace: armada/g' | kubectl create -n armada -f -
	sleep 10
	kubectl wait --for=condition=Ready pods -l  app.kubernetes.io/name=prometheus-operator -n armada --timeout=180s
	kubectl apply -n armada -f ./config/samples/prometheus.yaml

##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest
GOTESTSUM ?= $(LOCALBIN)/gotestsum
MOCKGEN ?= $(LOCALBIN)/mockgen
KIND    ?= $(LOCALBIN)/kind
HELMIFY ?= $(LOCALBIN)/helmify
GORELEASER ?= $(LOCALBIN)/goreleaser
## Tool Versions
KUSTOMIZE_VERSION ?= v4.5.7
CONTROLLER_TOOLS_VERSION ?= v0.10.0

KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"
.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	test -s $(LOCALBIN)/kustomize || { curl -Ss $(KUSTOMIZE_INSTALL_SCRIPT) | bash -s -- $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN); }

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

.PHONY: gotestsum
gotestsum: $(GOTESTSUM) ## Download gotestsum locally if necessary.
$(GOTESTSUM): $(LOCALBIN)
	test -s $(LOCALBIN)/gotestsum || GOBIN=$(LOCALBIN) go install gotest.tools/gotestsum@v1.8.2

.PHONY: mockgen
mockgen: $(MOCKGEN) ## Download mockgen locally if necessary.
$(MOCKGEN): $(LOCALBIN)
	test -s $(LOCALBIN)/mockgen || GOBIN=$(LOCALBIN) go install github.com/golang/mock/mockgen@v1.6.0

.PHONY: kind
kind: $(KIND)
$(KIND): $(LOCALBIN)
	test -s $(LOCALBIN)/kind || GOBIN=$(LOCALBIN) go install sigs.k8s.io/kind@v0.14.0

.PHONY: helmify
helmify: $(HELMIFY)
$(HELMIFY): $(LOCALBIN)
	test -s $(LOCALBIN)/helmify || GOBIN=$(LOCALBIN) go install github.com/arttor/helmify/cmd/helmify@v0.3.22

.PHONY: goreleaser
goreleaser: $(GORELEASER)
$(GORELEASER): $(LOCALBIN)
	test -s $(LOCALBIN)/goreleaser || GOBIN=$(LOCALBIN) go install github.com/goreleaser/goreleaser@v1.11.5


.PHONY: bundle
bundle: manifests kustomize ## Generate bundle manifests and metadata, then validate generated files.
	operator-sdk generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	$(KUSTOMIZE) build config/manifests | operator-sdk generate bundle $(BUNDLE_GEN_FLAGS)
	operator-sdk bundle validate ./bundle

.PHONY: bundle-build
bundle-build: ## Build the bundle image.
	docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

.PHONY: bundle-push
bundle-push: ## Push the bundle image.
	$(MAKE) docker-push IMG=$(BUNDLE_IMG)

.PHONY: opm
OPM = ./bin/opm
opm: ## Download opm locally if necessary.
ifeq (,$(wildcard $(OPM)))
ifeq (,$(shell which opm 2>/dev/null))
	@{ \
	set -e ;\
	mkdir -p $(dir $(OPM)) ;\
	OS=$(shell go env GOOS) && ARCH=$(shell go env GOARCH) && \
	curl -sSLo $(OPM) https://github.com/operator-framework/operator-registry/releases/download/v1.23.0/$${OS}-$${ARCH}-opm ;\
	chmod +x $(OPM) ;\
	}
else
OPM = $(shell which opm)
endif
endif

# A comma-separated list of bundle images (e.g. make catalog-build BUNDLE_IMGS=example.com/operator-bundle:v0.1.0,example.com/operator-bundle:v0.2.0).
# These images MUST exist in a registry and be pull-able.
BUNDLE_IMGS ?= $(BUNDLE_IMG)

# The image tag given to the resulting catalog image (e.g. make catalog-build CATALOG_IMG=example.com/operator-catalog:v0.2.0).
CATALOG_IMG ?= $(IMAGE_TAG_BASE)-catalog:v$(VERSION)

# Set CATALOG_BASE_IMG to an existing catalog image tag to add $BUNDLE_IMGS to that image.
ifneq ($(origin CATALOG_BASE_IMG), undefined)
FROM_INDEX_OPT := --from-index $(CATALOG_BASE_IMG)
endif

# Build a catalog image by adding bundle images to an empty catalog using the operator package manager tool, 'opm'.
# This recipe invokes 'opm' in 'semver' bundle add mode. For more information on add modes, see:
# https://github.com/operator-framework/community-operators/blob/7f1438c/docs/packaging-operator.md#updating-your-existing-operator
.PHONY: catalog-build
catalog-build: opm ## Build a catalog image.
	$(OPM) index add --container-tool docker --mode semver --tag $(CATALOG_IMG) --bundles $(BUNDLE_IMGS) $(FROM_INDEX_OPT)

# Push the catalog image.
.PHONY: catalog-push
catalog-push: ## Push a catalog image.
	$(MAKE) docker-push IMG=$(CATALOG_IMG)

.PHONY: helm
HELM = ./bin/helm
OS=$(shell go env GOOS)
HELM_VERSION=helm-v3.11.0-$(OS)-$(ARCH)
HELM_ARCHIVE=$(HELM_VERSION).tar.gz

helm: ## Download helm locally if necessary.
ifeq (,$(wildcard $(HELM)))
ifeq (,$(shell which helm 2>/dev/null))
	@{ \
	set -e ;\
	mkdir -p $(dir $(HELM)) ;\
	mkdir -p ./download ;\
    cd download ;\
	echo $(OS) $(ARCH) $(HELM_VERSION) $(HELM_ARCHIVE) ;\
	curl -sSLo ./$(HELM_ARCHIVE) https://get.helm.sh/$(HELM_ARCHIVE) ;\
	tar -zxvf ./$(HELM_ARCHIVE) ;\
    cd .. ;\
	ln -s ./download/$(OS)-$(ARCH)/helm $(HELM) ;\
	chmod +x $(HELM) ;\
	}
else
HELM = $(shell which helm)
endif
endif

.PHONY: create-dev-cluster
create-dev-cluster:
	kind create cluster --name $(KIND_DEV_CLUSTER_NAME) --config hack/kind-config.yaml
	kubectl create namespace armada

# Setup dependencies for a local development environment
.PHONY: dev-setup
dev-setup: create-dev-cluster install-pulsar helm-install-postgres \
    helm-install-redis dev-install-prometheus-operator \
    install-cert-manager install-ingress-controller dev-setup-webhook-tls

.PHONY: dev-install-controller
dev-install-controller: go-release-build load-image deploy

.PHONY: dev-teardown
dev-teardown:
	kind delete cluster --name $(KIND_DEV_CLUSTER_NAME)

.PHONY: dev-run
dev-run: dev-setup install run

WEBHOOK_TLS_OUT_DIR=/tmp/k8s-webhook-server/serving-certs
.PHONY: dev-setup-webhook-tls
dev-setup-webhook-tls:
	mkdir -p $(WEBHOOK_TLS_OUT_DIR)
	openssl req -new -newkey rsa:4096 -x509 -sha256 -days 365 -nodes -config dev/tls/webhooks_csr.conf -out $(WEBHOOK_TLS_OUT_DIR)/tls.crt -keyout $(WEBHOOK_TLS_OUT_DIR)/tls.key

dev-remove-webhook-tls:
	rm $(WEBHOOK_TLS_OUT_DIR)/tls.{crt,key}
