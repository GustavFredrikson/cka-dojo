# Worked solution

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: processor
  namespace: dojo-workloads-exam
spec:
  replicas: 3
  selector:
    matchLabels:
      app: processor
  template:
    metadata:
      labels:
        app: processor
    spec:
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
              - matchExpressions:
                  - key: kubernetes.io/hostname
                    operator: In
                    values: [worker2]
      containers:
        - name: processor
          image: nginx:1.27-alpine
          resources:
            requests: {cpu: 100m, memory: 64Mi}
            limits: {cpu: 250m, memory: 128Mi}
```

```bash
kubectl apply -f processor.yaml
kubectl -n dojo-workloads-exam rollout status deployment/processor
```
