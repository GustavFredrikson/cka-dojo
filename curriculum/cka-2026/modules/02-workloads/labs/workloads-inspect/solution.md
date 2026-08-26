# Worked solution

```bash
kubectl -n dojo-workloads-inspect get deploy,rs,pod
kubectl -n dojo-workloads-inspect get pod -l app=web -o yaml
kubectl -n dojo-workloads-inspect get deploy web -o yaml
```

The Deployment owns a ReplicaSet, the ReplicaSet directly owns Pods, and the
Pod template imports ConfigMap `web-config`.
