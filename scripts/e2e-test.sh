#!/bin/bash
#
# End-to-end test for the Armada Operator.
#
# Builds the operator from source and brings up a full Armada install on a kind cluster
# via the operator (make kind-all-dev), then creates a queue, submits a job,
# and waits for that job to succeed.
# All of the logic lives in this script so it runs the same way locally and in CI;
# CI just calls `scripts/e2e-test.sh`.
#
# Usage:
#   scripts/e2e-test.sh                          # full run, deletes the cluster at the end
#   E2E_KEEP_CLUSTER=true scripts/e2e-test.sh    # keep the kind cluster for debugging
#   E2E_TIMEOUT=900 scripts/e2e-test.sh          # override job-wait timeout (seconds)
#
# Environment overrides:
#   E2E_QUEUE         queue name to create/submit to     (default: example)
#   E2E_JOBSET        job set id to watch                 (default: job-set-1)
#   E2E_TIMEOUT       seconds to wait for job success     (default: 600)
#   E2E_KEEP_CLUSTER  "true" to skip cluster teardown     (default: false)
#
set -euo pipefail

GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'
log() { echo -e "${GREEN}$1${NC}"; }
err() { echo -e "${RED}$1${NC}" >&2; }

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

# `make get-armadactl` installs armadactl into bin/app; put it on PATH.
export PATH="$ROOT/bin/app:$PATH"

QUEUE="${E2E_QUEUE:-example}"
JOBSET="${E2E_JOBSET:-job-set-1}"
TIMEOUT="${E2E_TIMEOUT:-600}"
KEEP_CLUSTER="${E2E_KEEP_CLUSTER:-false}"

# retry_until <timeout_secs> <description> <cmd...>
# Runs cmd until it exits 0 or the timeout elapses, polling every 5s.
# If cmd's stderr reports the resource "already exists", that is treated as success.
# Each retry prints the elapsed time, and every ~30s it also prints a snapshot of the
# armada pods, so a stalled wait can be diagnosed straight from the log.
retry_until() {
  local timeout="$1" desc="$2"
  shift 2
  local err_file start now last_snap
  err_file="$(mktemp)"
  start=$(date +%s)
  last_snap=$start
  until "$@" 2>"${err_file}"; do
    cat "${err_file}" >&2
    if grep -qiE "already exists" "${err_file}"; then
      log "${desc}: already exists, continuing"
      break
    fi
    now=$(date +%s)
    if [ "$(( now - start ))" -ge "${timeout}" ]; then
      err "${desc}: did not succeed within ${timeout}s."
      rm -f "${err_file}"
      return 1
    fi
    if [ "$(( now - last_snap ))" -ge 30 ]; then
      last_snap=$now
      log "${desc}: still waiting ($(( now - start ))s/${timeout}s); armada pods:"
      kubectl get pods -n armada --no-headers 2>/dev/null \
        | awk '{printf "    %-58s %-7s %s\n", $1, $2, $3}' || true
    else
      log "${desc}: not ready yet ($(( now - start ))s/${timeout}s), retrying in 5s..."
    fi
    sleep 5
  done
  rm -f "${err_file}"
}

dump_diagnostics() {
  err "==================== E2E FAILED: diagnostics ===================="
  kubectl get pods -A || true
  kubectl get armadaservers,executors,schedulers,lookouts,eventingesters -A || true
  kubectl -n armada describe pods || true
  kubectl -n armada logs -l app=armada-server --tail=200 --all-containers || true
  kubectl -n armada logs -l app=armada-scheduler --tail=200 --all-containers || true
  kubectl -n armada logs -l app=armada-executor --tail=200 --all-containers || true
  kubectl -n armada-system logs deployment/armada-operator-controller-manager \
    -c manager --tail=200 || true
  kubectl get events -A --sort-by=.lastTimestamp | tail -100 || true
  err "================================================================"
}

cleanup() {
  rc=$?
  rm -f "${events:-}"
  if [ "$rc" -ne 0 ]; then
    dump_diagnostics
  fi
  if [ "$KEEP_CLUSTER" = "true" ]; then
    log "E2E_KEEP_CLUSTER=true -- leaving the kind cluster running."
  else
    log "Deleting kind cluster..."
    make kind-delete-cluster || true
  fi
  exit "$rc"
}
trap cleanup EXIT

log "==> Building the operator from source and bringing up Armada on kind (make kind-all-dev)..."
make kind-all-dev

# `make wait-for-armada` only waits for the armada-server pod to be Ready,
# but the `queue` table is created by the scheduler database migration,
# which can finish a little later.
# Retry until the API actually accepts queue creation before submitting.
log "==> Waiting for Armada to accept queue operations and creating queue '${QUEUE}'..."
retry_until 300 "create queue '${QUEUE}'" armadactl create queue "${QUEUE}"

# A job submitted before the scheduler has the executor's nodes is Rejected
# (JobFailedEvent cause=4, Cause_Rejected).
# Each scheduling cycle the scheduler logs the capacity of every pool it knows about,
# so wait until pool "default" has non-zero CPU (the executor's nodes are registered).
# After that a submit is accepted and scheduled, which keeps this deterministic
# and avoids a submit/reject retry loop.
# This also subsumes waiting for the executor pod: the scheduler only reports capacity
# once the executor is up and has registered.
log "==> Waiting for the scheduler to register the executor's capacity..."
scheduler_has_capacity() {
  kubectl logs -n armada -l app=armada-scheduler --tail=200 2>/dev/null \
    | grep -qE "Scheduling on pool .* with capacity \(memory=[0-9]+,cpu=[1-9]"
}
retry_until 600 "scheduler has executor capacity" scheduler_has_capacity

# CreateQueue persists to Postgres,
# but the submit API serves from a queue cache that refreshes on an interval,
# so retry submit until the queue becomes visible.
log "==> Submitting hello-world job to job set '${JOBSET}'..."
retry_until 120 "submit job to queue '${QUEUE}'" armadactl submit dev/quickstart/example-job.yaml

# The set holds a single job,
# so `watch --exit-if-inactive` returns as soon as that job is terminal.
# watch exits 0 even on failure, so inspect the raw event stream ourselves.
log "==> Waiting up to ${TIMEOUT}s for the job to finish..."
events="$(mktemp)"
timeout "${TIMEOUT}" armadactl watch --raw --exit-if-inactive "${QUEUE}" "${JOBSET}" | tee "${events}" || true

if grep -q 'JobSucceededEvent' "${events}"; then
  log "==> E2E PASSED: job in job set '${JOBSET}' succeeded."
  exit 0
fi
failed="$(grep 'JobFailedEvent' "${events}" | tail -1 || true)"
if [ -n "${failed}" ]; then
  err "Job in job set '${JOBSET}' failed: ${failed}"
  exit 1
fi
err "Job in job set '${JOBSET}' did not reach a terminal state within ${TIMEOUT}s."
exit 1
