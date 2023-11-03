# Lookout

Lookout is the Armada UI which shows information about jobs which are running in our Armada cluster.

Let's run the following command to start Lookout:
```bash
kubectl port-forward svc/armada-lookout-v1 8080 --namespace armada
open http://localhost:8080
```
