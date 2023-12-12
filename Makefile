# VERSION defines the project version for the bundle.
# Update this value when you upgrade the version of your project.
# To re-generate a bundle for another specific version without changing the standard setup, you can:
# - use the VERSION as arg of the bundle target (e.g make bundle VERSION=0.0.2)
# - use environment variables to overwrite this value (e.g export VERSION=0.0.2)
VERSION ?= 0.0.1

# IMAGE_TAG_BASE defines the docker.io namespace and part of the image name for remote images.
# This variable is used to construct full image tags for bundle and catalog images.
#
# For example, running 'make bundle-build bundle-push catalog-build catalog-push' will build and push both
# armadaproject.io/armada-operator-bundle:$VERSION and armadaproject.io/armada-operator-catalog:$VERSION.
IMAGE_TAG_BASE ?= armadaproject.io/armada-operator

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

##@ Codegen

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

.PHONY: generate-helm-chart
generate-helm-chart: manifests kustomize helmify ## Generate Helm chart from Kustomize manifests
	$(KUSTOMIZE) build config/default | $(HELMIFY) -crd-dir charts/armada-operator
	./hack/fix-helmify.sh

##@ Linting

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

##@ Tests

.PHONY: test-unit
test-unit: manifests generate fmt vet gotestsum ## Run unit tests.
	$(GOTESTSUM) -- ./internal/controller/... -coverprofile operator.out

.PHONY: test-integration
test-integration: manifests generate fmt vet gotestsum envtest ## Run integration tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" $(GOTESTSUM) -- ./test/... ./api/...

# Integration test without Ginkgo colorized output and control chars, for logging purposes
.PHONY: test-integration-debug
test-integration-debug: manifests generate fmt vet gotestsum envtest ## Run integration tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test -v ./test/... ./api/... --coverprofile integration.out -args --ginkgo.no-color

##@ Build

.PHONY: build
build: generate fmt vet ## Build manager binary.
	go build -o bin/manager main.go

.PHONY: go-releaser-build
go-releaser-build: goreleaser ## Build using GoReleaser
	$(GORELEASER) build --clean

.PHONY: go-releaser-snapshot
go-releaser-snapshot: goreleaser ## Build a snapshot release using GoReleaser
	$(GORELEASER) release --skip-publish --skip-sign --skip-sbom --clean --snapshot

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	go run ./cmd/main.go

run-no-webhook: manifests generate fmt vet ## Run a controller from your host without webhooks.
	ENABLE_WEBHOOKS=false go run ./cmd/main.go

# If you wish built the manager image targeting other platforms you can use the --platform flag.
# (i.e. docker build --platform linux/arm64 ). However, you must enable docker buildKit for it.
# More info: https://docs.docker.com/develop/develop-images/build_enhancements/
.PHONY: docker-build
docker-build: ## Build Operator Docker image
	docker build -t ${IMG} .

.PHONY: docker-push
docker-push: ## Push Operator Docker image
	docker push ${IMG}

.PHONY: load-image-kind
load-image-kind: ## Load Operator Docker image into kind cluster
	kind load docker-image --name $(KIND_DEV_CLUSTER_NAME) ${IMG}

# PLATFORMS defines the target platforms for  the manager image be build to provide support to multiple
# architectures. (i.e. make docker-buildx IMG=myregistry/mypoperator:0.0.1). To use this option you need to:
# - able to use docker buildx . More info: https://docs.docker.com/build/buildx/
# - have enable BuildKit, More info: https://docs.docker.com/develop/develop-images/build_enhancements/
# - be able to push the image for your registry (i.e. if you do not inform a valid value via IMG=<myregistry/image:<tag>> than the export will fail)
# To properly provided solutions that supports more than one platform you should use this option.
PLATFORMS ?= linux/arm64,linux/amd64,linux/s390x,linux/ppc64le
.PHONY: docker-buildx
docker-buildx: test ## Build and push Operator Docker image with cross-platform support
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

.PHONY: create-dev-cluster
create-dev-cluster: ## Create a local development cluster using kind
	kind create cluster --name $(KIND_DEV_CLUSTER_NAME) --config hack/kind-config.yaml
	kubectl create namespace armada
	kubectl create namespace data

.PHONY: kind-create
kind-create: kind ## Create a local development cluster using kind using config from hack/kind-config.yaml
	kind create cluster --config hack/kind-config.yaml

.PHONY: delete-dev-cluster
delete-dev-cluster: ## Delete the local development cluster using kind
	kind delete cluster --name $(KIND_DEV_CLUSTER_NAME)

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
deploy-to-kind: dev-setup docker-build load-image deploy ## Deploy controller to the local development cluster using kind

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/default | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: dev-install-controller
dev-install-controller: go-release-build load-image deploy

.PHONY: dev-run
dev-run: dev-setup install run

WEBHOOK_TLS_OUT_DIR=/tmp/k8s-webhook-server/serving-certs
.PHONY: dev-setup-webhook-tls
dev-setup-webhook-tls: ## Generate TLS certificates for webhook server
	mkdir -p $(WEBHOOK_TLS_OUT_DIR)
	openssl req -new -newkey rsa:4096 -x509 -sha256 -days 365 -nodes -config dev/tls/webhooks_csr.conf -out $(WEBHOOK_TLS_OUT_DIR)/tls.crt -keyout $(WEBHOOK_TLS_OUT_DIR)/tls.key

dev-remove-webhook-tls: ## Remove TLS certificates for webhook server
	rm $(WEBHOOK_TLS_OUT_DIR)/tls.{crt,key}

##@ External Dependencies

## Kubernetes Dependencies
CERT_MANAGER_MANIFEST ?= "https://github.com/cert-manager/cert-manager/releases/download/v1.6.3/cert-manager.yaml"
.PHONY: install-cert-manager
install-cert-manager: ## Install cert-manager
	kubectl apply -f ${CERT_MANAGER_MANIFEST}

.PHONY: uninstall-cert-manager
uninstall-cert-manager: ## Uninstall cert-manager
	kubectl delete -f ${CERT_MANAGER_MANIFEST}

INGRESS_MANIFEST ?= "https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml"
.PHONY: install-ingress-controller
install-ingress-controller: ## Install ingress controller
	kubectl apply -f ${INGRESS_MANIFEST}
.PHONY: uninstall-ingress-controller

uninstall-ingress-controller: ## Uninstall ingress controller
	kubectl delete -f ${INGRESS_MANIFEST}

.PHONY: helm-repos
helm-repos: helm ## Add helm repos
	$(HELM) repo add bitnami https://charts.bitnami.com/bitnami
	$(HELM) repo add apache https://pulsar.apache.org/charts
	$(HELM) repo add dandydev https://dandydeveloper.github.io/charts
	$(HELM) repo update

.PHONY: install-pulsar
install-pulsar: helm-repos ## Install pulsar
	$(HELM) install pulsar apache/pulsar -f dev/quickstart/pulsar.values.yaml --namespace data

.PHONY: helm-install-postgres
helm-install-postgres: helm-repos ## Install postgres
	helm install postgres bitnami/postgresql -f dev/quickstart/postgres.values.yaml --create-namespace --namespace data

.PHONY: helm-install-redis
helm-install-redis: helm-repos ## Install redis
	$(HELM) install redis dandydev/redis-ha -f dev/quickstart/redis.values.yaml --create-namespace --namespace data

PROMETHEUS_OPERATOR_VERSION=v0.62.0
.PHONY: dev-install-prometheus-operator
dev-install-prometheus-operator: ## Install prometheus operator
	curl -sL https://github.com/prometheus-operator/prometheus-operator/releases/download/${PROMETHEUS_OPERATOR_VERSION}/bundle.yaml | sed -e 's/namespace: default/namespace: armada/g' | kubectl create -n armada -f -
	sleep 10
	kubectl wait --for=condition=Ready pods -l  app.kubernetes.io/name=prometheus-operator -n armada --timeout=180s
	kubectl apply -n armada -f ./config/samples/prometheus.yaml

# Setup dependencies for a local development environment
.PHONY: dev-setup
dev-setup: dev-install-prometheus-operator install-pulsar \
    helm-install-redis helm-install-postgres 		      \
    install-cert-manager install-ingress-controller dev-setup-webhook-tls

##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN): ## Create local bin directory if necessary.
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
CONTROLLER_TOOLS_VERSION ?= v0.13.0

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
	test -s $(LOCALBIN)/gotestsum || GOBIN=$(LOCALBIN) go install gotest.tools/gotestsum@v1.11.0

.PHONY: mockgen
mockgen: $(MOCKGEN) ## Download mockgen locally if necessary.
$(MOCKGEN): $(LOCALBIN)
	test -s $(LOCALBIN)/mockgen || GOBIN=$(LOCALBIN) go install github.com/golang/mock/mockgen@v1.6.0

.PHONY: kind
kind: $(KIND) ## Download kind locally if necessary.
$(KIND): $(LOCALBIN)
	test -s $(LOCALBIN)/kind || GOBIN=$(LOCALBIN) go install sigs.k8s.io/kind@v0.14.0

.PHONY: helmify
helmify: $(HELMIFY) ## Download helmify locally if necessary.
$(HELMIFY): $(LOCALBIN)
	test -s $(LOCALBIN)/helmify || GOBIN=$(LOCALBIN) go install github.com/arttor/helmify/cmd/helmify@v0.4.6

.PHONY: goreleaser
goreleaser: $(GORELEASER) ## Download GoReleaser locally if necessary.
$(GORELEASER): $(LOCALBIN)
	test -s $(LOCALBIN)/goreleaser || GOBIN=$(LOCALBIN) go install github.com/goreleaser/goreleaser@v1.21.2

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
