# Worked solution

```bash
kubectl -n dojo-workloads-guided get pods
kubectl -n dojo-workloads-guided logs deployment/web --previous
kubectl -n dojo-workloads-guided patch deployment web --type=json \
  -p='[{"op":"remove","path":"/spec/template/spec/containers/0/command"}]'
kubectl -n dojo-workloads-guided rollout status deployment/web
```
