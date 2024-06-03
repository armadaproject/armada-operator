# Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN): ## Create local bin directory if necessary.
	mkdir -p $(LOCALBIN)
# LOCALBIN_TOOLING refers to the directory where tooling binaries are installed.
LOCALBIN_TOOLING ?= $(LOCALBIN)/tooling
$(LOCALBIN_TOOLING): ## Create local bin directory for tooling if necessary.
	mkdir -p $(LOCALBIN_TOOLING)
# LOCALBIN_APP refers to the directory where application binaries are installed.
LOCALBIN_APP ?= $(LOCALBIN)/app
$(LOCALBIN_APP): ## Create local bin directory for app if necessary.
	mkdir -p $(LOCALBIN_APP)

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.29

# VERSION defines the latest semver of the operator.
VERSION ?= $(shell git describe --tags --abbrev=0)

# PLATFORMS defines the target platforms for the operator image.
PLATFORMS ?= linux/amd64,linux/arm64
# IMAGE_REGISTRY defines the registry where the operator image will be pushed.
IMAGE_REGISTRY ?= gresearch
# IMAGE_NAME defines the name of the operator image.
IMAGE_NAME := armada-operator
# IMAGE_REPO defines the image repository and name where the operator image will be pushed.
IMAGE_REPO ?= $(IMAGE_REGISTRY)/$(IMAGE_NAME)
# GIT_TAG defines the git tag of the operator image.
GIT_TAG ?= $(shell git describe --tags --dirty --always)
# IMAGE_TAG defines the name and tag of the operator image.
IMAGE_TAG ?= $(IMAGE_REPO):$(GIT_TAG)

# KIND_CLUSTER_NAME defines the name of the kind cluster to be created.
KIND_CLUSTER_NAME=armada

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

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
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-25s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Codegen

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: mock
mock: mockgen ## Generate mock client for k8sclient
	$(RM) test/k8sclient/mock_client.go
	mockgen -destination=test/k8sclient/mock_client.go -package=k8sclient "github.com/armadaproject/armada-operator/test/k8sclient" Client

.PHONY: generate-helm-chart
generate-helm-chart: manifests kustomize helmify ## Generate Helm chart from Kustomize manifests
	$(KUSTOMIZE) build config/default | $(HELMIFY) -crd-dir charts/armada-operator

generate-crd-ref-docs: crd-ref-docs ## Generate CRD reference documentation
	$(CRD_REF_DOCS) crd-ref-docs --source-path=./api --config=./hack/crd-ref-docs-config.yaml --renderer=markdown --output-path=./dev/crd

##@ Linting

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: lint
lint: golangci-lint ## Run golangci-lint against code.
	$(GOLANGCI_LINT) run

.PHONY: lint-fix
lint-fix: golangci-lint ## Run golangci-lint against code and fix issues.
	$(GOLANGCI_LINT) run --fix

##@ Tests

.PHONY: test-unit
test-unit: manifests generate gotestsum ## Run unit tests.
	$(GOTESTSUM) -- ./internal/controller/... -coverprofile operator.out

.PHONY: test-integration
test-integration: manifests generate gotestsum envtest ## Run integration tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" $(GOTESTSUM) -- ./test/... ./api/...

# Integration test without Ginkgo colorized output and control chars, for logging purposes
.PHONY: test-integration-debug
test-integration-debug: manifests generate fmt vet gotestsum envtest ## Run integration tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test -v ./test/... ./api/... --coverprofile integration.out -args --ginkgo.no-color

##@ Build

.PHONY: build
build: generate ## Build armada operator binary.
	go build -o $(LOCALBIN_APP)/armada-operator cmd/main.go

.PHONY: build-linux-amd64
build-linux-amd64: generate ## Build armada operator binary for linux/amd64.
	GOOS=linux GOARCH=amd64 go build -o $(LOCALBIN_APP)/armada-operator cmd/main.go

.PHONY: goreleaser-build
goreleaser-build: goreleaser ## Build using GoReleaser
	$(GORELEASER) build --skip validate --clean

.PHONY: goreleaser-snapshot
goreleaser-snapshot: goreleaser ## Build a snapshot release using GoReleaser
	$(GORELEASER) release --skip-publish --skip-sign --skip-sbom --clean --snapshot

##@ Run

.PHONY: run
run: manifests generate install dev-setup-webhook-tls ## Run operator locally with admission webhooks
	WEBHOOK_CERT_DIR=$(WEBHOOK_TLS_OUT_DIR) go run ./cmd/main.go

.PHONY: run-no-webhook
run-no-webhook: manifests generate install ## Run operator locally without admission webhooks
	ENABLE_WEBHOOKS=false go run ./cmd/main.go

WEBHOOK_TLS_OUT_DIR=/tmp/k8s-webhook-server/serving-certs
.PHONY: dev-setup-webhook-tls
dev-setup-webhook-tls: ## Generate TLS certificates for webhook server
	mkdir -p $(WEBHOOK_TLS_OUT_DIR)
	openssl req -new -newkey rsa:4096 -x509 -sha256 -days 365 -nodes -config hack/tls/webhooks_csr.conf -out $(WEBHOOK_TLS_OUT_DIR)/tls.crt -keyout $(WEBHOOK_TLS_OUT_DIR)/tls.key

.PHONY: dev-remove-webhook-tls
dev-remove-webhook-tls: ## Remove TLS certificates for webhook server
	rm $(WEBHOOK_TLS_OUT_DIR)/tls.{crt,key}

##@ Package

.PHONY: docker-build
docker-build: build-linux-amd64 ## Build Armada Operator Docker image using buildx.
	docker buildx build 		\
		--file Dockerfile 		\
		--tag ${IMAGE_TAG} 		\
		--output type=docker	\
		$(LOCALBIN_APP)

.PHONY: docker-push
docker-push: build-linux-amd64 ## Push Armada Operator Docker image using buildx.
	docker buildx build 		\
		--file Dockerfile 		\
		--platform=$(PLATFORMS) \
		--tag ${IMAGE_TAG} 		\
		--push 					\
		$(LOCALBIN_APP)

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: kind-all
kind-all: kind-create-cluster kind-deploy ## Create a kind cluster and deploy the operator.

.PHONY: kind-create-cluster
kind-create-cluster: kind ## Create a kind cluster using config from hack/kind-config.yaml.
	$(KIND) create cluster --config hack/kind-config.yaml --name $(KIND_CLUSTER_NAME)

.PHONY: kind-load-image
kind-load-image: docker-build ## Load Operator Docker image into kind cluster.
	$(KIND) load docker-image --name $(KIND_CLUSTER_NAME) ${IMAGE_TAG}

.PHONY: kind-deploy
kind-deploy: kind-load-image install install-cert-manager wait-for-cert-manager ## Deploy operator in a kind cluster after building & loading the image
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMAGE_TAG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

.PHONY: kind-delete-cluster
kind-delete-cluster: ## Delete the local development cluster using kind.
	$(KIND) delete cluster --name $(KIND_CLUSTER_NAME)

.PHONY: install
install: kustomize manifests ## Install Armada Operator CRDs.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

.PHONY: uninstall
uninstall: kustomize manifests ## Uninstall Armada Operator CRDs.
	$(KUSTOMIZE) build config/crd | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy
deploy: manifests kustomize ## Deploy Armada Operator using Kustomize.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMAGE_TAG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

helm-install: helm ## Install latest released Armada Operator using Helm from gresearch Helm registry.
	$(HELM) upgrade --install armada-operator gresearch/armada-operator --create-namespace --namespace armada-system

helm-install-local: kind-load-image helm ## Build latest Armada Operator and install using Helm. Should only be used for local development in kind cluster.
	$(HELM) upgrade --install armada-operator charts/armada-operator --create-namespace --namespace armada-system --set controllerManager.manager.image.tag=$(GIT_TAG)

helm-uninstall: helm ## Uninstall operator using Helm.
	$(HELM) uninstall armada-operator --namespace armada-operator

.PHONY: undeploy
undeploy: ## Delete Armada Operator Kubernetes resources.
	$(KUSTOMIZE) build config/default | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

##@ External Dependencies

.PHONY: install-armada-deps
install-armada-deps: helm-repos helm-install-kube-prometheus-stack helm-install-postgres helm-install-pulsar helm-install-redis ## Install required Armada dependencies (Prometheus, PostgreSQL, Pulsar, Redis).

CERT_MANAGER_MANIFEST ?= "https://github.com/cert-manager/cert-manager/releases/download/v1.14.5/cert-manager.yaml"
.PHONY: install-cert-manager
install-cert-manager: ## Install cert-manager.
	kubectl apply -f ${CERT_MANAGER_MANIFEST}

.PHONY: wait-for-cert-manager
wait-for-cert-manager: ## Wait for cert-manager to be ready.
	kubectl wait --for=condition=Available --timeout=600s -n cert-manager deployments --all

install-and-wait-cert-manager: install-cert-manager wait-for-cert-manager ## Install and wait for cert-manager to be ready.

.PHONY: uninstall-cert-manager
uninstall-cert-manager: ## Uninstall cert-manager.
	kubectl delete -f ${CERT_MANAGER_MANIFEST}

INGRESS_MANIFEST ?= "https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml"
.PHONY: install-ingress-controller
install-ingress-controller: ## Install ingress controller.
	kubectl apply -f ${INGRESS_MANIFEST}
.PHONY: uninstall-ingress-controller

uninstall-ingress-controller: ## Uninstall ingress controller.
	kubectl delete -f ${INGRESS_MANIFEST}

.PHONY: helm-repos
helm-repos: helm ## Add helm repos for external dependencies.
	$(HELM) repo add jetstack https://charts.jetstack.io
	$(HELM) repo add bitnami https://charts.bitnami.com/bitnami
	$(HELM) repo add apache https://pulsar.apache.org/charts
	$(HELM) repo add dandydev https://dandydeveloper.github.io/charts
	$(HELM) repo update

CERT_MANAGER_VERSION ?= v1.14.5
.PHONY: helm-install-cert-manager
helm-install-cert-manager:
	$(HELM) upgrade --install cert-manager jetstack/cert-manager \
	  --create-namespace 				 			   \
	  --namespace cert-manager 			 			   \
	  --version $(CERT_MANAGER_VERSION)  			   \
	  --set installCRDs=true

.PHONY: helm-install-pulsar
helm-install-pulsar: helm-repos ## Install Apache Pulsar using Helm.
	$(HELM) upgrade --install pulsar apache/pulsar --values dev/quickstart/pulsar.values.yaml --create-namespace --namespace data

.PHONY: helm-uninstall-pulsar
helm-uninstall-pulsar: ## Uninstall Apache Pulsar using Helm.
	$(HELM) uninstall pulsar --namespace data

.PHONY: helm-install-postgres
helm-install-postgres: helm-repos ## Install PostgreSQL using Helm.
	helm upgrade --install postgres bitnami/postgresql --values dev/quickstart/postgres.values.yaml --create-namespace --namespace data

.PHONY: helm-uninstall-postgres
helm-uninstall-postgres: ## Uninstall PostgreSQL using Helm.
	$(HELM) uninstall postgres --namespace data

.PHONY: helm-install-redis
helm-install-redis: helm-repos ## Install Redis using Helm.
	$(HELM) upgrade --install redis-ha dandydev/redis-ha --values dev/quickstart/redis.values.yaml --create-namespace --namespace data

.PHONY: helm-uninstall-redis
helm-uninstall-redis: ## Uninstall Redis using Helm.
	$(HELM) uninstall redis --namespace data

.PHONY: helm-install-kube-prometheus-stack
helm-install-kube-prometheus-stack: helm-repos ## Install kube-prometheus-stack using Helm.
	$(HELM) upgrade --install kube-prometheus-stack prometheus-community/kube-prometheus-stack --create-namespace --namespace monitoring

.PHONY: helm-uninstall-kube-prometheus-stack
helm-uninstall-kube-prometheus-stack: ## Uninstall kube-prometheus-stack using Helm.
	$(HELM) uninstall kube-prometheus-stack --namespace monitoring

##@ Build Dependencies

## Tool Binaries
KUSTOMIZE ?= $(LOCALBIN_TOOLING)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN_TOOLING)/controller-gen
ENVTEST ?= $(LOCALBIN_TOOLING)/setup-envtest
GOTESTSUM ?= $(LOCALBIN_TOOLING)/gotestsum
MOCKGEN ?= $(LOCALBIN_TOOLING)/mockgen
KIND    ?= $(LOCALBIN_TOOLING)/kind
HELMIFY ?= $(LOCALBIN_TOOLING)/helmify
GORELEASER ?= $(LOCALBIN_TOOLING)/goreleaser
CRD_REF_DOCS ?= $(LOCALBIN_TOOLING)/crd-ref-docs
GOLANGCI_LINT ?= $(LOCALBIN_TOOLING)/golangci-lint

KUSTOMIZE_VERSION ?= v5.4.2
KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"
.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN_TOOLING)
	test -s $(KUSTOMIZE) || { curl -Ss $(KUSTOMIZE_INSTALL_SCRIPT) | bash -s -- $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN_TOOLING); }

CONTROLLER_TOOLS_VERSION ?= v0.15.0
.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(CONTROLLER_GEN) || GOBIN=$(LOCALBIN_TOOLING) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN_TOOLING)
	test -s $(ENVTEST) || GOBIN=$(LOCALBIN_TOOLING) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

GOTESTSUM_VERSION ?= v1.11.0
.PHONY: gotestsum
gotestsum: $(GOTESTSUM) ## Download gotestsum locally if necessary.
$(GOTESTSUM): $(LOCALBIN_TOOLING)
	test -s $(GOTESTSUM) || GOBIN=$(LOCALBIN_TOOLING) go install gotest.tools/gotestsum@$(GOTESTSUM_VERSION)

MOCKGEN_VERSION ?= v1.6.0
.PHONY: mockgen
mockgen: $(MOCKGEN) ## Download mockgen locally if necessary.
$(MOCKGEN): $(LOCALBIN_TOOLING)
	test -s $(MOCKGEN) || GOBIN=$(LOCALBIN_TOOLING) go install github.com/golang/mock/mockgen@$(MOCKGEN_VERSION)

KIND_VERSION ?= v0.23.0
.PHONY: kind
kind: $(KIND) ## Download kind locally if necessary.
$(KIND): $(LOCALBIN_TOOLING)
	test -s $(KIND) || GOBIN=$(LOCALBIN_TOOLING) go install sigs.k8s.io/kind@$(KIND_VERSION)

HELMIFY_VERSION ?= v0.4.13
.PHONY: helmify
helmify: $(HELMIFY) ## Download helmify locally if necessary.
$(HELMIFY): $(LOCALBIN_TOOLING)
	test -s $(HELMIFY) || GOBIN=$(LOCALBIN_TOOLING) go install github.com/arttor/helmify/cmd/helmify@$(HELMIFY_VERSION)

GORELEASER_VERSION ?= v1.26.2
.PHONY: goreleaser
goreleaser: $(GORELEASER) ## Download GoReleaser locally if necessary.
$(GORELEASER): $(LOCALBIN_TOOLING)
	test -s $(GORELEASER) || GOBIN=$(LOCALBIN_TOOLING) go install github.com/goreleaser/goreleaser@$(GORELEASER_VERSION)

CRD_REF_DOCS_VERSION ?= v0.0.12
.PHONY: crd-ref-docs
crd-ref-docs: $(CRD_REF_DOCS) ## Download crd-ref-docs locally if necessary.
$(CRD_REF_DOCS): $(LOCALBIN_TOOLING)
	test -s $(CRD_REF_DOCS) || GOBIN=$(LOCALBIN_TOOLING) go install github.com/elastic/crd-ref-docs@$(CRD_REF_DOCS_VERSION)

GOLANGCI_LINT_VERSION ?= v1.59.0
.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN_TOOLING)
	test -s $(GOLANGCI_LINT) || GOBIN=$(LOCALBIN_TOOLING) go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

.PHONY: helm
HELM = ./bin/helm
OS=$(shell go env GOOS)
HELM_VERSION=helm-v3.15.0-$(OS)-$(ARCH)
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
