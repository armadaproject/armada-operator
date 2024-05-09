# Armada Operator

First, we need to install the Armada Operator, so we can install Armada by creating the Armada components CRDs.
Let's run the following commands to install the Armada Operator:

```bash
helm repo add gresearch https://g-research.github.io/charts
helm install armada-operator gresearch/armada-operator --namespace armada-system --create-namespace
```
