# Give a namespace sensible defaults

## Working it out

A LimitRange has no imperative `kubectl create` form, so this one is written
by hand. The two field names are easy to swap by accident:

- `default` — the default **limits**
- `defaultRequest` — the default **requests**

`type: Container` applies them per container. `type: Pod` sets bounds on the
Pod total and has no defaulting behaviour at all, which is a common wrong turn.

The defaults are applied by an admission plugin when the Pod is created. They
are not retroactive: a Pod that already exists keeps whatever it had, and
editing the LimitRange later changes nothing about it. Order is the whole
lesson here — LimitRange first, Pod second.

## Fixing it

```yaml
apiVersion: v1
kind: LimitRange
metadata: {name: defaults, namespace: dojo-limitrange-build}
spec:
  limits:
    - type: Container
      defaultRequest: {cpu: 100m, memory: 64Mi}
      default: {cpu: 250m, memory: 128Mi}
```

```bash
kubectl apply -f limitrange.yaml
kubectl -n dojo-limitrange-build run probe --image=busybox:1.36 -- sleep 3600
```

`kubectl run` writes no `resources` block, which is exactly what the task
wants — the values must come from admission, not from you.

If you created the Pod first and it has no resources, delete and recreate it:

```bash
kubectl -n dojo-limitrange-build delete pod probe
kubectl -n dojo-limitrange-build run probe --image=busybox:1.36 -- sleep 3600
```

## Checking before you grade

```bash
kubectl -n dojo-limitrange-build get pod probe \
  -o jsonpath='{.spec.containers[0].resources}'; echo
```

You should see all four values, none of which you typed. If the block is
empty, the Pod predates the LimitRange.
