#!/bin/bash

STATEFULSETS=(
    "postgresql"
    "pulsar-bookie"
    "pulsar-broker"
    "pulsar-proxy"
    "pulsar-recovery"
    "pulsar-toolset"
    "pulsar-zookeeper"
    "redis-ha-server"
)

NAMESPACE=data

TIMEOUT=1200
INTERVAL=10

check_statefulsets() {
    for ss in "${STATEFULSETS[@]}"; do
        echo "Checking StatefulSet $ss..."

        READY=$(kubectl get statefulset "$ss" -o jsonpath='{.status.readyReplicas}' --namespace $NAMESPACE)

        if [ -z "$READY" ] || [ "$READY" -le 0 ]; then
            echo "StatefulSet $ss does not have at least 1 ready replicas: $READY"
            return 1
        fi
    done
    return 0
}

start_time=$(date +%s)

while true; do
    if check_statefulsets; then
        echo "All StatefulSets have at least 1 ready replicas."
        exit 0
    fi

    elapsed_time=$(($(date +%s) - start_time))
    if [ "$elapsed_time" -ge "$TIMEOUT" ]; then
        echo "Timeout reached: Not all StatefulSets are ready within ${TIMEOUT} seconds."
        exit 1
    fi

    echo "Waiting for StatefulSets to be ready... (elapsed time: ${elapsed_time} seconds)"
    sleep $INTERVAL
done
