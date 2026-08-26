# Worked solution

```bash
kubectl -n dojo-workloads-probe edit deployment web
# Set readinessProbe.httpGet.port to 80.
kubectl -n dojo-workloads-probe rollout status deployment/web
```
