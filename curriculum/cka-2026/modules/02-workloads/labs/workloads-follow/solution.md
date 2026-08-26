# Worked solution

```bash
kubectl -n dojo-workloads-follow create deployment web --image=nginx:1.26-alpine --replicas=2
kubectl -n dojo-workloads-follow rollout status deployment/web
kubectl -n dojo-workloads-follow scale deployment web --replicas=3
kubectl -n dojo-workloads-follow set image deployment/web nginx=nginx:1.27-alpine
kubectl -n dojo-workloads-follow rollout status deployment/web
kubectl -n dojo-workloads-follow get deployment,replicaset,pod
```
