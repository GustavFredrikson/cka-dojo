# Worked solution

The task contains the complete replacement manifest. The important diagnostic
commands are:

```bash
kubectl describe node worker2
kubectl -n dojo-scheduling-follow describe pod web
kubectl -n dojo-scheduling-follow get events --sort-by=.lastTimestamp
```

The node selector leaves only `worker2` as a candidate. Its
`dedicated=training:NoSchedule` taint then filters that candidate out. The
toleration permits—rather than forces—the Pod to use that node; the selector
continues to provide the attraction.
