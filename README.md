# armada-operator
- [armada-operator](#armada-operator)
  - [How it works](#how-it-works)
  - [Installing Armada Operator](#installing-armada-operator)
    - [Prerequisites](#prerequisites)
    - [Quickstart](#quickstart)
    - [Manual setup](#manual-setup)
    - [Installing Armada](#installing-armada)
    - [Applying PriorityClass](#applying-priorityclass)
    - [Installing `armadactl`](#installing-armadactl)
  - [Using `armadactl`](#using-armadactl)
  - [Migrating](#migrating)
    - [Migrating to v0.11 and beyond](#migrating-to-v011-and-beyond)
  - [Further documentation](#further-documentation)
  - [Supported Armada component configurations](#supported-armada-component-configurations)
  - [Developing locally](#developing-locally)
    - [Starting a development cluster](#starting-a-development-cluster)
    - [Building and installing a development cluster](#building-and-installing-a-development-cluster)
    - [Running the Armada Operator locally](#running-the-armada-operator-locally)
    - [Stopping the development cluster](#stopping-the-development-cluster)
  - [Contributing](#contributing)
  - [Testing](#testing)
  - [FAQ](#faq)
    - [`kube-prometheus-stack` is not installing](#kube-prometheus-stack-is-not-installing)
  - [License](#license)

[![GoReport Widget]][GoReport Status]
[![Latest Release](https://img.shields.io/github/v/release/armadaproject/armada-operator?include_prereleases)](https://github.com/armadaproject/armada-operator/releases/latest)

[GoReport Widget]: https://goreportcard.com/badge/github.com/armadaproject/armada-operator
[GoReport Status]: https://goreportcard.com/report/github.com/armadaproject/armada-operator

Armada Operator is a Kubernetes-native Operator for simpler installation of [Armada](https://armadaproject.io).
This project introduces [Custom Resource Definitions (CRDs)](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/) for Armada services and provides a controller to manage the lifecycle of these services.

## How it works

This project aims to follow the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/).

It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/),
which provide a reconcile function responsible for synchronising resources until the desired state is reached on the cluster.

## Installing Armada Operator

### Prerequisites

Before you start, make sure you have the following installed:

* [kubectl](https://kubernetes.io/docs/tasks/tools/)
* [helm](https://helm.sh/docs/intro/install/)
* [Go v1.21+](https://golang.org/doc/install) - required for the Quickstart `make kind-all` target

### Quickstart

To start immediately with Armada and the Operator, run the following `make` target:

```bash
make kind-all
```

This command runs the following `make` targets:

* `kind-create-cluster` - creates a `kind` cluster using the configuration from `hack/kind-config.yaml`
* `install-and-wait-cert-manager` - installs `cert-manager` and waits for it to be ready
* `helm-repos` - adds necessary Helm repositories for external dependencies
* `helm-install` - installs the latest released Armada Operator using Helm from the gresearch Helm registry
* `install-armada-deps` - installs required Armada dependencies such as Prometheus, PostgreSQL, Pulsar and Redis using Helm
* `wait-for-armada-deps` - waits for all Armada dependencies to be ready
* `create-armada-namespace` - creates the Armada namespace `armada`
* `apply-armada-crs` - applies the CRDs of all Armada components using kubectl
* `create-armadactl-config` - creates the armadactl configuration (`~/.armadactl.yaml` file, pointing to `localhost:30002` as per the quickstart guide
* `apply-default-priority-class` - applies the default priority class required by Armada for all jobs
* `get-armadactl` - downloads the armadactl binary for interacting with the Armada API

### Manual setup

You’ll need a Kubernetes cluster to run the Armada Operator. You can use
[Kubernetes in Docker (KIND)](https://sigs.k8s.io/kind) to run a local cluster for testing, or you
can run against a remote cluster.

To install the latest release of Armada Operator in your cluster, run the following commands:

```bash
# Add the G-Research Helm repository
helm repo add gresearch https://g-research.github.io/charts
# Install the Armada Operator
helm install armada-operator gresearch/armada-operator --namespace armada-system --create-namespace
```

This installs the Armada Operator in the `armada-system` namespace.

### Installing Armada

When installing Armada, you first need to install the Armada external dependencies:

* [kube-prometheus-stack](https://github.com/prometheus-community/helm-charts/tree/main/charts/kube-prometheus-stack)
* [Pulsar](https://pulsar.apache.org/)
* [Redis](https://redis.io/)
* [Postgres](https://www.postgresql.org/)

You can install the required Armada dependencies either manually or by running the following `make` target:

```bash
make install-armada-deps
```

**Note:** You need to wait for all dependencies to be running before proceeding to the next step.

Then run the following command to install Armada components using the CRs provided in `dev/quickstart/armada-crs.yaml`:

```bash
# Create armada namespace
kubectl create namespace armada
# Install Armada components
kubectl apply -n armada -f dev/quickstart/armada-crs.yaml
```

`dev/quickstart/armada-crs.yaml` uses NodePort Services for exposing Armada components For example, the Lookout UI at `30000`, the Armada [gRPC](en.wikipedia.org/wiki/GRPC) API at `30001` and the Armada REST API at `30002`.

Once every Armada service is deployed, you should have a fully functional installation of Armada.

**Note:** This example, created for use with the `kind` config defined at `hack/kind-config.yaml`, is for demonstration purposes only. Do not use it in production.

### Applying PriorityClass

Armada also requires a default PriorityClass to be set for all jobs. You can apply the default PriorityClass by running the following command:

```bash
kubectl apply -f dev/quickstart/priority-class.yaml
```

### Installing `armadactl`

To install `armadactl`, run the following `make` target:

```bash
make get-armadactl
```

Alternatively, download it from the [GitHub release page](https://github.com/armadaproject/armada/releases/latest).

Create a configuration file for `armadactl` by running the following command:

```bash
make create-armadactl-config
```

Alternatively, create a configuration file manually in `~/.armadactl.yaml` with the following content:

```yaml
currentContext: main
contexts:
  main:
    # URL of the Armada gRPC API
    armadaUrl: localhost:30002 # This uses the NodePort configured in the Quickstart guide
```

## Using `armadactl`

Create a `queue` called `example` by running:

```bash
armadactl create queue example
```

Submit a job to the `example` queue by running:

```bash
armadactl submit dev/quickstart/example-job.yaml
```

Check the status of your job in the Lookout UI by visiting `http://localhost:30000` in your browser. (This assumes that you installed Armada using the Quickstart guide and exposed it via a NodePort service.)

## Migrating

### Migrating to v0.11 and beyond

Since v0.11, Armada Scheduler requires you to configure `permissionGroupsMapping`.

Make sure the `applicationConfig` field in the Armada Scheduler CRD includes the `permissionGroupsMapping` field.

Quickstart example that allows anonymous auth:

```yaml
auth:
  anonymousAuth: true
  permissionGroupMapping:
    execute_jobs: ["everyone"]
```

## Further documentation

To install Armada using the Armada Operator and interact with the Armada API,
[see the Quickstart guide](./dev/quickstart/README.md) and follow the runbooks in the `dev/runbooks/` directory.

For more info on Armada, see the [Armada website](https://armadaproject.io) and the GitHub repository [armadaproject/armada](https://github.com/armadaproject/armada)

To understand the minimal configuration required to deploy Armada services, [see the `armada-crs.yaml`](./dev/quickstart/armada-crs.yaml) file.

For advanced usage, see the Armada CRD reference docs in the `dev/crd/` directory.

## Supported Armada component configurations

Each Armada Operator CRD supports a `.spec.applicationConfig` field, which injects configuration into the Armada component, for example:

* [Armada Server](https://pkg.go.dev/github.com/armadaproject/armada/internal/armada/configuration#ArmadaConfig)
* [Armada Executor](https://pkg.go.dev/github.com/armadaproject/armada/internal/executor/configuration#ApplicationConfiguration)
* [Armada Scheduler](https://pkg.go.dev/github.com/armadaproject/armada/internal/scheduler/configuration#Configuration)
* [Armada Scheduler Ingester](https://pkg.go.dev/github.com/armadaproject/armada/internal/scheduleringester#Configuration)
* [Armada Lookout](https://pkg.go.dev/github.com/armadaproject/armada/internal/lookoutv2/configuration#LookoutV2Config)
* [Armada Lookout Ingester](https://pkg.go.dev/github.com/armadaproject/armada/internal/lookoutingesterv2/configuration#LookoutIngesterV2Configuration)
* [Armada Event Ingester](https://pkg.go.dev/github.com/armadaproject/armada/internal/eventingester/configuration#EventIngesterConfiguration)
* [Armada Binoculars](https://pkg.go.dev/github.com/armadaproject/armada/internal/binoculars/configuration#BinocularsConfig)

## Developing locally

This section assumes you have a `kind` cluster named `armada` running on your machine (it will appear as `kind-armada` in your kubeconfig).

See the Makefile for more commands to help you with your development, or run `make help` for a list of available commands.

Your controller will automatically use the current context in your kubeconfig file (whatever cluster `kubectl cluster-info` shows).

### Starting a development cluster

This section assumes you have [KIND](https://sigs.k8s.io/kind) installed.

If you do not have a Kubernetes cluster to test against, you can start one using the following command:

```bash
make kind-create-cluster
```

### Building and installing a development cluster

Run the following `make` command:

```bash
make kind-deploy
```
This command does the following:

* builds the `armada-operator` binary for `linux/amd64`
* builds the `armada-operator` Docker image
* loads the Docker image into the `kind` cluster
* installs each CRD supported by the `armada-operator` on the cluster
* installs the `armada-operator` on the cluster using `kustomize`

### Running the Armada Operator locally

To run the operator locally, use one of the following commands:

```bash
# Run the operator locally with webhooks enabled
make run
# Run the operator locally with webhooks disabled
make run-no-webhook
```

### Stopping the development cluster

To stop the development cluster:

```bash
make kind-delete-cluster
```

This will totally destroy your development Kind cluster.

## Contributing

You can contribute bug-reports or ideas for enhancements via GitHub's issue system. 

Code contributions are also welcome. When submitting a pull request, make sure:

* it references a relevant issue
* all CI checks pass

## Testing

You should test contributions thoroughly before you ask for a review. At a minimum, the following should all run without error:

```bash
# Lint code using golangci-lint.
make lint
# Run unit tests.
make test-unit
# Run integration tests.
make test-integration
```

Add and change appropriate unit and integration tests to ensure your changes are covered by automated tests and appear to be correct.

## FAQ

### `kube-prometheus-stack` is not installing

If you get the following error, upgrade your Helm version to `v3.16.2` or later:

```bash
Error: template: kube-prometheus-stack/templates/prometheus/prometheus.yaml:262:11: executing "kube-prometheus-stack/templates/prometheus/prometheus.yaml" at <ne .Values.prometheus.prometheusSpec.scrapeConfigNamespaceSelector nil>: error calling ne: uncomparable type map[string]interface {}: map[]
```

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

