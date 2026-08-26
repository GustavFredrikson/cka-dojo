# Worked solution

```bash
kubectl -n dojo-workloads-rollout set image deployment/web web=nginx:1.27-alpine
kubectl -n dojo-workloads-rollout rollout status deployment/web
kubectl -n dojo-workloads-rollout rollout history deployment/web
kubectl -n dojo-workloads-rollout rollout undo deployment/web
kubectl -n dojo-workloads-rollout rollout status deployment/web
```
