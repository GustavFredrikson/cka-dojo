# Worked solution

## Part 1: the immutable field

The obvious move does not work:

```bash
kubectl patch priorityclass dojo-critical -p '{"value":100000}'
```

```
The PriorityClass "dojo-critical" is invalid: value: Forbidden: may not be changed in an update
```

`value` is immutable. There is no flag, no `--force` on patch, and no
subresource that helps. The only route is delete and recreate:

```bash
kubectl delete priorityclass dojo-critical
kubectl create -f - <<'EOF'
apiVersion: scheduling.k8s.io/v1
kind: PriorityClass
metadata:
  name: dojo-critical
value: 100000
globalDefault: false
description: "Revenue-path workloads. Preempts batch."
EOF
```

`kubectl replace --force -f dojo-critical.yaml` does the same thing in one
command -- it deletes and recreates rather than updating, which is exactly
what is needed here and is worth knowing for every other immutable field you
meet.

The consequence to be aware of: **priority is copied into a Pod at admission**
and stored in `spec.priority`. Existing Pods keep the number they were
admitted with. So recreating the class changes nothing about what is already
running, and everything about what is admitted next. Nothing needs restarting
for the new value to matter.

## Part 2: the Deployment

An extended resource goes in `limits`, not `requests`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: payments
  namespace: dojo-priority
spec:
  replicas: 1
  selector:
    matchLabels: {app: payments}
  template:
    metadata:
      labels: {app: payments}
    spec:
      priorityClassName: dojo-critical
      containers:
        - name: payments
          image: busybox:1.36
          command: ["sleep", "3600"]
          resources:
            limits:
              dojo.cka/slots: "1"
```

Extended resources are not overcommittable: the request is always set equal to
the limit, so specifying only `limits` is the normal form. They also must be
integers -- `"1"`, never `500m`.

## What the scheduler does

Both slots are held by `batch`, so `payments` does not fit anywhere. Watch:

```bash
kubectl -n dojo-priority get pods -w
kubectl -n dojo-priority describe pod <payments-pod> | tail -20
```

The events tell the story:

```
Warning  FailedScheduling  0/3 nodes are available: 1 node(s) had untolerated
                           taint(s), 2 Insufficient dojo.cka/slots.
Normal   Preempted         Preempted by pod 27196eb6-2fb6-44cf-a104-cb2225b10254
                           on node worker1
```

Two things about reading these. The `Preempted` event is on the **victim**, not
on the Pod that caused it, and it identifies the winner by UID rather than by
name -- so to find out who evicted what, look at the events of the Pod that
disappeared. And you will also see `preemption: not eligible due to a
terminating pod on the nominated node` against `payments` for a few seconds:
that is not a failure, it is the scheduler waiting for the victim it already
chose to finish going away.

The scheduler could not place the Pod, looked for lower-priority Pods whose
eviction would make it fit, found one `batch` Pod at priority 1000 against
`payments` at 100000, and deleted it. `payments` then schedules into the freed
slot. The `batch` ReplicaSet immediately creates a replacement, which stays
`Pending` -- there is no room, and that is the correct outcome, not a
secondary failure.

If `dojo-critical` had still been 100, none of this happens: 100 is below
`dojo-batch`'s 1000, so `payments` would be the *lower* priority Pod and would
simply sit `Pending` forever. That is the whole reason the value matters.

## Things that would have been the wrong answer

- Scaling `batch` down, or deleting one of its Pods. It makes `payments`
  schedule and proves nothing; the task forbids it because the point is that
  the scheduler does this for you.
- `preemptionPolicy: Never` on `dojo-critical`. That means "schedule me
  ahead of lower-priority Pods in the queue, but never evict anyone". A
  reasonable choice for important-but-not-urgent work, and here it would leave
  `payments` Pending.

## Checking before you grade

```bash
kubectl get priorityclass dojo-critical
kubectl -n dojo-priority get deployment batch payments
kubectl -n dojo-priority get pods -o wide
```

`dojo-critical` at 100000; `payments` 1/1 on `worker1`; `batch` showing 1/2
with one Pod `Pending`.
