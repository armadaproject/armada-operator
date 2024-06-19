# Demo Armada Setup

## Prerequisites

### Demo Kubernetes Cluster requirements
* [cert-manager](https://cert-manager.io/) - cert-manager is a Kubernetes add-on to automate the management and issuance of TLS certificates from various issuing sources.

### Namespaces

#### Data namespace

All data storages will be deployed in the `data` namespace.

Run the following command to create the namespace:
```bash
kubectl create namespace data
```

#### Armada namespaces

All Armada components will be deployed in the `armada` namespace.

Run the following command to create the namespace:
```bash
kubectl create namespace armada
```

### Pulsar

Apache Pulsar is the event bus and message broker used by Armada components.

Run the following commands to install Pulsar:
```bash
helm repo add apache https://pulsar.apache.org/charts
helm install pulsar apache/pulsar -f dev/quickstart/pulsar.values.yaml --namespace data
```

### Redis

Redis is a key-value store used by Armada components for caching.

Run the following commands to install Redis:
```bash
helm repo add dandydev https://dandydeveloper.github.io/charts
helm install redis dandydev/redis-ha -f dev/quickstart/redis.values.yaml --namespace data
```

### Postgres

Postgres is a relational database used by Armada components.

Run the following commands to install Postgres:
```bash
helm repo add bitnami https://charts.bitnami.com/bitnami
helm install postgres bitnami/postgresql -f dev/quickstart/postgres.values.yaml --namespace data
```

### NGINX Controller

`ingress-nginx` is an Ingress controller for Kubernetes using NGINX as a reverse proxy and load balancer.

Run the following commands to install NGINX Controller:
```bash
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
helm install nginx ingress-nginx/ingress-nginx --namespace kube-system
```

### Kube Prometheus Stack

The `kube-prometheus-stack` chart provides a collection of Kubernetes manifests, Grafana dashboards, and Prometheus rules combined with documentation and scripts to provide easy to operate end-to-end Kubernetes cluster monitoring with Prometheus using the Prometheus Operator.

Run the following commands to install Kube Prometheus Stack:
```bash
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm install kube-prometheus-stack prometheus-community/kube-prometheus-stack --namespace monitoring --create-namespace
```

### Kube State Metrics

`kube-state-metrics` is a simple service that listens to the Kubernetes API server and generates metrics about the state of the objects.

Run the following commands to install Kube State Metrics:
```bash
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm install kube-state-metrics prometheus-community/kube-state-metrics --namespace monitoring
```

## Armada Operator

Armada Operator is used to deploy and manage Armada components.

Run the following commands to install Armada Operator:
```bash
helm install armada-operator ./charts/armada-operator --namespace armada --create-namespace
```

## Armada Components

As we are using the Armada Operator to deploy Armada components, we will install Armada components by creating the required CRDs.

Run the following commands to install Armada components:
```bash
kubectl apply -f dev/quickstart/armada-crs.yaml --namespace armada
```

## Testing Armada

First we need to create a default `PriorityClass` for Armada jobs.

Run the following commands to create the `armada-default` PriorityClass:
```bash
kubectl apply -f dev/quickstart/priority-class.yaml
```

## Minimal setup

Minimal setup for Armada which could only accept jobs would consist only of Armada Server and Executor deployed in the same cluster.

Run the following command to install the minimal setup:
```bash
kubectl apply -f dev/quickstart/minimal-setup.yaml
```
