#!/bin/bash

set -eo pipefail

GREEN='\033[0;32m'
NC='\033[0m'

log() {
    echo -e "${GREEN}$1${NC}"
}

armadactl-retry() {
  for i in {1..60}; do
    if ! armadactl "$@"; then
      sleep 1
    fi
  done
}

log "Running e2e tests..."

log "Creating kind cluster..."
make kind-create-cluster

log "Installing cert-manager..."
make install-and-wait-cert-manager

log "Building latest version of the operator and installing it via Helm..."
make helm-install-local

for i in {1..5}; do
    kubectl get pods -n armada-system
    kubectl describe deployment armada-operator-controller-manager -n armada-system
    sleep 15
    kubectl logs deployment/armada-operator-controller-manager -n armada-system
done

log "Waiting for Armada Operator pod to be ready..."
kubectl wait --for='condition=Ready' pods -l 'app.kubernetes.io/name=armada-operator' --timeout=600s --namespace armada-system

log "Creating test queue..."
armadactl-retry create queue test

log "Submitting job..."
armadactl-retry submit dev/quickstart/example-job.yaml

log "Deleting kind cluster..."
make kind-delete-cluster
