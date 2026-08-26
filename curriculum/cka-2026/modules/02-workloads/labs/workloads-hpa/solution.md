# Worked solution

```bash
kubectl -n dojo-workloads-hpa autoscale deployment web --min=2 --max=6 --cpu=60%
kubectl -n dojo-workloads-hpa get hpa web
```
