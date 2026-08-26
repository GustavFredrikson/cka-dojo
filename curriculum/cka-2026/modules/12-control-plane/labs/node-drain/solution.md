# Worked solution

```bash
kubectl cordon worker1
kubectl drain worker1 --ignore-daemonsets --delete-emptydir-data
kubectl -n dojo-node-drain get pods -o wide
```
