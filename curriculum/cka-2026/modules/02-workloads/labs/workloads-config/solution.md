# Worked solution

```bash
kubectl -n dojo-workloads-config create configmap web-settings --from-literal=APP_MODE=production
kubectl -n dojo-workloads-config create secret generic api-credentials --from-literal=token=swordfish
kubectl -n dojo-workloads-config create deployment web --image=nginx:1.27-alpine --replicas=2
kubectl -n dojo-workloads-config set env deployment/web --from=configmap/web-settings
kubectl -n dojo-workloads-config set env deployment/web --from=secret/api-credentials
kubectl -n dojo-workloads-config rollout status deployment/web
```
