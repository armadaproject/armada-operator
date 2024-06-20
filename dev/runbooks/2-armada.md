# Armada

Now we need to apply the Armada components CRDs and the Operator will install & configure the Armada cluster.
Let's run the following commands to create the Armada resources:
```bash
kubectl create namespace armada
kubectl apply -f dev/quickstart/armada-crs.yaml --namespace armada
```
