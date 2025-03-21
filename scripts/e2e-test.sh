#!/bin/bash

set -eo pipefail

GREEN='\033[0;32m'
NC='\033[0m'

log() {
    echo -e "${GREEN}$1${NC}"
}

armadactl-retry() {
  for i in {1..60}; do
    if armadactl "$@"; then
      return 0
    fi
    sleep 1
  done
  echo "armadactl command failed after 60 attempts" >&2
  exit 1
}

log "Running e2e tests..."

log "Creating test environment cluster..."
make kind-all-local

log "Creating test queue..."
armadactl-retry create queue example

log "Submitting job..."
armadactl-retry submit dev/quickstart/example-job.yaml

log "Deleting kind cluster..."
make kind-delete-cluster
