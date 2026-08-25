# Worked solution

Exporting first is a quick way to preserve the existing fields:

```bash
kubectl -n dojo-scheduling-guided get pod batch -o yaml > /tmp/batch.yaml
```

Remove generated metadata and `status`, add this under `spec`, then replace
the Pod:

```yaml
tolerations:
  - key: dedicated
    operator: Equal
    value: training
    effect: NoSchedule
```

```bash
kubectl -n dojo-scheduling-guided delete pod batch
kubectl apply -f /tmp/batch.yaml
kubectl -n dojo-scheduling-guided get pod batch -o wide
```
