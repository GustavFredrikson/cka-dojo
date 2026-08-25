# Worked solution

```bash
kubectl -n dojo-scheduling-inspect get pod report -o wide
kubectl -n dojo-scheduling-inspect get pod report -o yaml
kubectl get nodes --show-labels
```

The Pod requires `dojo-tier=special`. Only `worker2` has that label, so
`worker1` is filtered out and the binding appears as `spec.nodeName: worker2`.
