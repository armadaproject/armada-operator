# Armada Jobs

Armada Jobs are a superset of a Kubernetes PodSpec with additional Armada-specific fields.
```yaml
queue: queue-a
jobSetId: job-set-1
jobs:
  - priority: 0
    podSpec:
      containers:
        - name: sleeper
          image: alpine:latest
          command:
            - sh
          args:
            - -c
            - sleep $(( (RANDOM % 60) + 10 ))
```

Let's submit an Armada job in our Armada cluster by running the following command:
```bash
armadactl submit ./dev/quickstart/job.yaml
```
