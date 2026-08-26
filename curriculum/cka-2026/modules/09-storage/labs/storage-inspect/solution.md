# Worked solution

```bash
kubectl get pv dojo-storage-inspect -o wide
kubectl -n dojo-storage-inspect get pvc data -o wide
kubectl describe pv dojo-storage-inspect
kubectl -n dojo-storage-inspect describe pvc data
```
