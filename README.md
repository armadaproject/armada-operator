# armada-operator

[![GoReport Widget]][GoReport Status]
[![Latest Release](https://img.shields.io/github/v/release/armadaproject/armada-operator?include_prereleases)](https://github.com/armadaproject/armada-operator/releases/latest)

[GoReport Widget]: https://goreportcard.com/badge/github.com/armadaproject/armada-operator
[GoReport Status]: https://goreportcard.com/report/github.com/armadaproject/armada-operator

Armada Operator is a Kubernetes-native Operator for simpler installation of [Armada](https://armadaproject.io).
This project introduces CRDs for Armada services and provides a controller to manage the lifecycle of these services.

## How it works
This project aims to follow the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)

It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/)
which provides a reconcile function responsible for synchronizing resources until the desired state is reached on the cluster

## Installation

You’ll need a Kubernetes cluster to run the Armada Operator. You can use
[KIND](https://sigs.k8s.io/kind) to run a local cluster for testing, or you
can run against a remote cluster.

To install the latest release of Armada Operator in your cluster, run the following commands:
```bash
# Add the G-Research Helm repository
helm repo add gresearch https://gresearch.github.io/armada-operator
# Install the Armada Operator
helm install armada-operator gresearch/armada-operator --namespace armada-system --create-namespace
```

The Armada Operator will be installed in the `armada-system` namespace.

### Installing Armada

In order to install Armada, first you will need to install the Armada external dependencies:
* [Pulsar](https://pulsar.apache.org/)
* [Redis](https://redis.io/)
* [Postgres](https://www.postgresql.org/)

You can install the required Armada dependencies either manually or by running the following `make` target:
```bash
make install-armada-deps
```

**Note:** You will need to wait for all dependencies to be running before proceeding to the next step.

Finally:
```bash
# Create armada namespace
kubectl create namespace armada
# Install Armada components
kubectl apply -n armada -f dev/quickstart/armada-crds.yaml
```

Which will deploy samples of each CRD. Once every Armada service is deployed,
you should have a fully functional installation of Armada.

## Documentation

For a step-by-step guide on how to install Armada using the Armada Operator and interact with the Armada API,
please read the Quickstart guide in the [documentation](./dev/quickstart/README.md) and follow runbooks in the `dev/runbooks/` directory.

For more info on Armada, please visit the [Armada website](https://armadaproject.io) and the GitHub repository [armadaproject/armada](https://github.com/armadaproject/armada)

## Local development

This section assumes you have a `kind` cluster named `armada` running on your machine (it will appear as `kind-armada` in your kubeconfig).

Check out the Makefile for more commands to help you with your development or run `make help` for a list of available commands.

**Note:** Your controller will automatically use the current context in your 
kubeconfig file (i.e. whatever cluster `kubectl cluster-info` shows).

### Start a Development Cluster

This section assumes you have [KIND](https://sigs.k8s.io/kind) installed.

If you do not have a Kubernetes cluster to test against, you can start one using the following command:
```bash
make kind-create-cluster
```

### Build & install from scratch

Run the following `make` command:
```bash
make kind-deploy
```
This command will do the following:
- Build the `armada-operator` binary for `linux/amd64`
- Build the `armada-operator` Docker image
- Load the Docker image into the `kind` cluster
- Install each CRD supported by the `armada-operator` on the cluster
- Install the `armada-operator` on the cluster using `kustomize`

### Run locally

In order to run the operator locally, you can use one of the following commands:
```bash
# Run the operator locally with webhooks enabled
make run
# Run the operator locally with webhooks disabled
make run-no-webhook
```

### Stop the Development Cluster

To stop the development cluster:
```bash
make kind-delete-cluster
```

This will totally destroy your development Kind cluster.

## Contributing

Please feel free to contribute bug-reports or ideas for enhancements via 
GitHub's issue system. 

Code contributions are also welcome. When submitting a pull-request please 
ensure it references a relevant issue as well as making sure all CI checks 
pass.

## Testing

Please test contributions thoroughly before requesting reviews. At a minimum:
```bash
# Lint code using golangci-lint.
make lint
# Run unit tests.
make test-unit
# Run integration tests.
make test-integration
```
should all succeed without error. 

Add and change appropriate unit and integration tests to ensure your changes 
are covered by automated tests and appear to be correct.

## License

Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

