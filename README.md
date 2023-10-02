# armada-operator
`armada-operator` is a small [go](https://go.dev/) project to automate the 
installation (and eventually management) of a fully-functional 
[Armada](https://github.com/armadaproject/armada) deployment
to a [Kubernetes](https://kubernetes.io/) cluster using the Kubernetes 
[operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/).

## Description
Armada is a multi-Kubernetes batch job scheduler. This operator aims to make
Armada easy to deploy and, well, operate in a Kubernetes cluster. 

## Quickstart

Want to start hacking right away?

You’ll need a Kubernetes cluster to run the operator. You can use 
[KIND](https://sigs.k8s.io/kind) to run a local cluster for testing, or you 
can run against a remote cluster.  

**Note:** Your controller will automatically use the current context in your 
kubeconfig file (i.e. whatever cluster `kubectl cluster-info` shows).

### Start a Development Cluster

This section assumes you have [KIND](https://sigs.k8s.io/kind) installed.

If you do not have a Kubernetes cluster to test against, you can start one using the following command:
```bash
make create-dev-cluster
```

Run the following command to install all Armada external dependencies (Apache Pulsar, Redis, PostgreSQL, NGINX, Prometheus)
```bash
make dev-setup
```

Then:
```bash
make dev-install-controller
```
Which will:
- install each CRD supported by the armada-operator on the cluster
- create a pod inside the kind cluster running the armada-operator controllers

**Note:** You may need to wait for some services (like Pulsar) to finish 
coming up to proceed to the next step. Check the status of 
the cluster with `$ kubectl get -n armada pods`.

Finally:
```bash
kubectl apply -n armada -f $(REPO_ROOT)/dev/quickstart/armada-crds.yaml
```

Which will deploy samples of each CRD. Once every Armada service is deployed,
you should have a fully functional install of Armada running.

To stop the development cluster:
```bash
make dev-teardown
```

This will totally destroy your development Kind cluster. 

## Getting Started

### Running on a Cluster

1. Build and push your image to the location specified by `IMG`:
	
```bash
make docker-build docker-push IMG=<some-registry>/armada-operator:tag
```
	
2. Deploy the controller to the cluster with the image specified by `IMG`:

```bash
make deploy IMG=<some-registry>/armada-operator:tag
```

3. Install Instances of Custom Resources:

```bash
kubectl apply -f config/samples/
```

### Uninstall CRDs
To delete the CRDs from the cluster:

```bash
make uninstall
```

### Undeploy controller
UnDeploy the controller to the cluster:

```bash
make undeploy
```

## Contributing

Please feel free to contribute bug-reports or ideas for enhancements via 
GitHub's issue system. 

Code contributions are also welcome. When submitting a pull-request please 
ensure it references a relevant issue as well as making sure all CI checks 
pass.

### Test All Changes

Please test contributions thoroughly before requesting reviews. At a minimum:
```bash
make test
make test-integration
make lint
```
should all succeed without error. 

Add and change appropriate unit and integration tests to ensure your changes 
are covered by automated tests and appear to be correct.

### How it works
This project aims to follow the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)

It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/) 
which provides a reconcile function responsible for synchronizing resources untile the desired state is reached on the cluster 

### Test It Out
1. Install the CRDs into the cluster:

```bash
make install
```

2. Run your controller (this will run in the foreground, so switch to a new terminal if you want to leave it running):

```bash
make run
```

**NOTE:** You can also run this in one step by running: `make install run`

### Modifying the API definitions
If you are editing the API definitions, generate the manifests such as CRs or CRDs using:

```bash
make manifests
```

**NOTE:** Run `make --help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

