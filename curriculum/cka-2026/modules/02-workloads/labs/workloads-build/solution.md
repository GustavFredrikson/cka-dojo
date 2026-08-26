# Worked solution

Generate a starting point, then add resources under the container:

```bash
kubectl -n dojo-workloads-build create deployment api \
  --image=nginx:1.27-alpine --replicas=3 --dry-run=client -o yaml > /tmp/api.yaml
```

```yaml
resources:
  requests: {cpu: 25m, memory: 32Mi}
  limits: {cpu: 100m, memory: 64Mi}
```

Apply it and verify with `kubectl rollout status deployment/api`.
