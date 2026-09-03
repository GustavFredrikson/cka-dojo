# Read what the cluster is actually using

## Working it out

Two numbers describe every container and they rarely agree. `kubectl top`
measures; `kubectl describe node` accounts for what was reserved.

```bash
kubectl -n dojo-observability-top top pods
```

`grinder` sits at roughly its 200m ceiling — it runs a busy loop, so the
kernel throttles it exactly at the limit. `dozer` sleeps and reports close to
`0m`.

Now look at what each one took out of the node's budget:

```bash
kubectl -n dojo-observability-top get pods -o custom-columns=\
NAME:.metadata.name,CPU_REQ:.spec.containers[0].resources.requests.cpu
```

`dozer` requests `400m` and uses nothing. `grinder` requests `50m` and uses
four times that. Both are normal-looking Deployments; only the pairing of the
two views shows which one is wasting capacity.

Confirm on the node itself:

```bash
kubectl describe node worker1 | sed -n '/Allocated resources/,/Events/p'
```

The percentages there are requests against allocatable — not usage. A node can
show 95% allocated and near-zero measured load.

## Why it matters

The scheduler only ever reads `requests`. `dozer` makes 400m of the node
unavailable to anything else while doing nothing with it, and that is what
causes a later Pod to sit Pending on a cluster that `kubectl top` says is idle.

## Checking before you grade

`kubectl -n dojo-observability-top top pods` must return rows. If it reports
that the metrics API is unavailable, give metrics-server a minute — it needs a
sampling window before a newly started Pod appears.
