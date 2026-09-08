# Nothing in the cluster can resolve a name

In namespace `dojo-coredns-down`, Pod `dnsprobe` cannot resolve the `web`
Service:

```
nslookup: can't resolve 'web.dojo-coredns-down.svc.cluster.local'
```

`web` itself is healthy, and so is `dnsprobe`. Every other namespace in the
cluster has the same problem.

Find what is broken and restore cluster DNS to the state kubeadm configured:
two backing Pods, reachable through the Service that every Pod points at.
