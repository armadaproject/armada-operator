# Joining a new Kubernetes cluster to the Armada cluster

Now that we have an Armada cluster running, let's join a new Kubernetes cluster to the Armada cluster.
We just need to install the Executor component in a Kubernetes cluster, and it will automagically join the Armada cluster.
```yaml
apiVersion: install.armadaproject.io/v1alpha1
kind: Executor
metadata:
  name: armada-executor
  namespace: armada
spec:
  image:
    repository: gresearch/armada-executor
    tag: 0.3.92
  applicationConfig:
    apiConnection:
      armadaUrl: server.demo.armadaproject.io:443
```

Let's run the following command to install the Executor in our Kubernetes cluster:
```bash
kubectl apply -f dev/quickstart/executor.yaml
```
