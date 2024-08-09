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

### Prerequisites

In order to start, make sure you have the following installed:
* [kubectl](https://kubernetes.io/docs/tasks/tools/)
* [helm](https://helm.sh/docs/intro/install/)
* [Go v1.21+](https://golang.org/doc/install) - required for the Quickstart `make kind-all` target

### Quickstart

To start immediately with Armada and the Operator, you can run the following `make` target:
```bash
make kind-all
```

This will run the following `make` targets:
1. `kind-create-cluster` - Creates a kind cluster using the configuration from hack/kind-config.yaml.
2. `install-and-wait-cert-manager` - Installs cert-manager and waits for it to be ready.
3. `helm-repos` - Adds necessary Helm repositories for external dependencies.
4. `helm-install` - Installs the latest released Armada Operator using Helm from the gresearch Helm registry.
5. `install-armada-deps` - Installs required Armada dependencies such as Prometheus, PostgreSQL, Pulsar, and Redis using Helm.
6. `wait-for-armada-deps` - Waits for all Armada dependencies to be ready.
7. `create-armada-namespace` - Creates the Armada namespace `armada`
8. `apply-armada-crs` - Applies the Custom Resource definitions (CRs) of all Armada components using kubectl.
9. `create-armadactl-config` - Creates the armadactl configuration (`~/.armadactl.yaml) file pointing to `localhost:30002` as per the quickstart guide.
10. `apply-default-priority-class` - Applies the default priority class required by Armada for all jobs.
11. `get-armadactl` - Downloads the armadactl binary for interacting with the Armada API.

### Manual setup

You’ll need a Kubernetes cluster to run the Armada Operator. You can use
[KIND](https://sigs.k8s.io/kind) to run a local cluster for testing, or you
can run against a remote cluster.

To install the latest release of Armada Operator in your cluster, run the following commands:
```bash
# Add the G-Research Helm repository
helm repo add gresearch https://g-research.github.io/charts
# Install the Armada Operator
helm install armada-operator gresearch/armada-operator --namespace armada-system --create-namespace
```

The Armada Operator will be installed in the `armada-system` namespace.

### Installing Armada

In order to install Armada, first you will need to install the Armada external dependencies:
* [kube-prometheus-stack](https://github.com/prometheus-community/helm-charts/tree/main/charts/kube-prometheus-stack)
* [Pulsar](https://pulsar.apache.org/)
* [Redis](https://redis.io/)
* [Postgres](https://www.postgresql.org/)

You can install the required Armada dependencies either manually or by running the following `make` target:
```bash
make install-armada-deps
```

**Note:** You will need to wait for all dependencies to be running before proceeding to the next step.

After that run the following command to install Armada components using the CRs provided in `dev/quickstart/armada-crs.yaml`
```bash
# Create armada namespace
kubectl create namespace armada
# Install Armada components
kubectl apply -n armada -f dev/quickstart/armada-crs.yaml
```

**Note:** `dev/quickstart/armada-crs.yaml` uses **NodePort** Services for exposing Armada components (Lookout UI @ `30000`, Armada gRPC API @ `30001` and Armada REST API @ `30002`).
This example is created to be used with the `kind` config defined at `hack/kind-config.yaml` for demonstration purposes only and should not be used in production.

Which will deploy CRs for each Armada component. Once every Armada service is deployed,
you should have a fully functional installation of Armada.

Armada requires a default PriorityClass to be set for all jobs. You can apply the default PriorityClass by running the following command:
```bash
kubectl apply -f dev/quickstart/priority-class.yaml
```

### Installing armadactl

To install `armadactl`, run the following `make` target:
```bash
make get-armadactl
```

Or download it from the [GitHub Release](https://github.com/armadaproject/armada/releases/latest) page for your platform.

Create a configuration file for `armadactl` by running the following command:
```bash
make create-armadactl-config
```

Or create a configuration file manually in `~/.armadactl.yaml` with the following content:
```yaml
currentContext: main
contexts:
  main:
    # URL of the Armada gRPC API
    armadaUrl: localhost:30002 # This uses the NodePort configured in the Quickstart guide
```

## Usage

Create a `queue` called `example` by running:
```bash
armadactl create queue example
```

Submit a job to the `example` queue by running:
```bash
armadactl submit dev/quickstart/example-job.yaml
```

Check the status of your job in the Lookout UI by visiting `http://localhost:30000` (assuming Armada was installed via the Quickstart guide and it is exposed via a NodePort service) in your browser.

## Documentation

For a step-by-step guide on how to install Armada using the Armada Operator and interact with the Armada API,
please read the Quickstart guide in the [documentation](./dev/quickstart/README.md) and follow runbooks in the `dev/runbooks/` directory.

For more info on Armada, please visit the [Armada website](https://armadaproject.io) and the GitHub repository [armadaproject/armada](https://github.com/armadaproject/armada)

For understanding the minimal configuration required to deploy Armada services, please refer to the [armada-crs.yaml](./dev/quickstart/armada-crs.yaml) file.

For advanced usage, please refer to the Armada CRD reference docs in `dev/crd/` directory.

Each Armada Operator CRD supports a field `.spec.applicationConfig` which injects configuration into the Armada component.
Here is a list of support Armada component configurations:
* [Armada Server](https://pkg.go.dev/github.com/armadaproject/armada/internal/armada/configuration#ArmadaConfig)
* [Armada Executor](https://pkg.go.dev/github.com/armadaproject/armada/internal/executor/configuration#ApplicationConfiguration)
* [Armada Scheduler](https://pkg.go.dev/github.com/armadaproject/armada/internal/scheduler/configuration#Configuration)
* [Armada Scheduler Ingester](https://pkg.go.dev/github.com/armadaproject/armada/internal/scheduleringester#Configuration)
* [Armada Lookout](https://pkg.go.dev/github.com/armadaproject/armada/internal/lookoutv2/configuration#LookoutV2Config)
* [Armada Lookout Ingester](https://pkg.go.dev/github.com/armadaproject/armada/internal/lookoutingesterv2/configuration#LookoutIngesterV2Configuration)
* [Armada Event Ingester](https://pkg.go.dev/github.com/armadaproject/armada/internal/eventingester/configuration#EventIngesterConfiguration)
* [Armada Binoculars](https://pkg.go.dev/github.com/armadaproject/armada/internal/binoculars/configuration#BinocularsConfig)

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

