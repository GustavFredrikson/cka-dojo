# Worked solution

## Working it out

```bash
kubectl -n dojo-scheduling-pending describe pod report
kubectl -n dojo-scheduling-pending get pod report -o yaml
kubectl get node worker2 --show-labels
kubectl describe node worker2
```

If the event says no node matches the selector, compare the required label
with `worker2`. If it names an untolerated taint, compare that taint's key,
value and effect with the Pod tolerations.

## Fixing it

For the selector variant, either add the intended label:

```bash
kubectl label node worker2 dojo-reserved=true
```

or replace the Pod with a selector that unambiguously requires `worker2`.

For the taint variant, either remove an unintended taint:

```bash
kubectl taint node worker2 dedicated:NoSchedule-
```

or replace the standalone Pod with a matching toleration. Any route that keeps
the required Pod on `worker2` is valid.

## Checking before you grade

```bash
kubectl -n dojo-scheduling-pending get pod report -o wide
kubectl -n dojo-scheduling-pending describe pod report
```
