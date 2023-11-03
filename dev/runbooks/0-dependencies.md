# Dependencies

Armada is a Kubernetes-native batch scheduler, and it requires a Kubernetes cluster for installation.
If you do not have a cluster, you can use [kind](https://kind.sigs.k8s.io/) to create a local cluster for testing purposes.
Let's run the following command to create a local cluster:
```bash
kind create cluster --name armada
```

Armada requires the following dependencies to be installed on the cluster:
* [PostgreSQL](https://www.postgresql.org/) - open source relational database
* [Redis](https://redis.io/) - open source, in-memory data store
* [Apache Pulsar](https://pulsar.apache.org/) - Cloud-Native, Distributed Messaging and Streaming
* [cert-manager](https://cert-manager.io/) - Kubernetes add-on to automate the management and issuance of TLS certificates from various issuing sources
* [NGINX Ingress Controller](https://kubernetes.github.io/ingress-nginx/) - Ingress controller that uses ConfigMap to store the NGINX configuration
* [Prometheus](https://prometheus.io/) - open-source systems monitoring and alerting toolkit
* OPTIONAL: [kube-state-metrics](https://github.com/kubernetes/kube-state-metrics) - add-on agent to generate and expose cluster-level metrics
* OPTIONAL: [Metrics Server](https://github.com/kubernetes-sigs/metrics-server) - add-on agent to collect resource metrics such as CPU and memory from nodes and pods

Let's run the following commands to install the required dependencies:
```bash
kubectl create namespace data

helm repo add jetstack https://charts.jetstack.io
helm upgrade --install               \
  cert-manager jetstack/cert-manager \
  --namespace cert-manager           \
  --create-namespace                 \
  --version v1.13.1                  \
  --set installCRDs=true

helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm upgrade --install kube-prometheus-stack prometheus-community/kube-prometheus-stack --namespace monitoring --create-namespace

helm repo add apache https://pulsar.apache.org/charts
helm upgrade --install pulsar apache/pulsar -f dev/quickstart/pulsar.values.yaml --namespace data

helm repo add dandydev https://dandydeveloper.github.io/charts
helm upgrade --install redis dandydev/redis-ha -f dev/quickstart/redis.values.yaml --namespace data

helm repo add bitnami https://charts.bitnami.com/bitnami
helm upgrade --install postgres bitnami/postgresql -f dev/quickstart/postgres.values.yaml --namespace data

helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
helm upgrade --install nginx ingress-nginx/ingress-nginx --namespace kube-system
```
