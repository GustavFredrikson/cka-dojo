# Resource usage and observability

## Mental model

```text
requests  = what the scheduler reserves   (planning)
limits    = what the kernel enforces      (runtime ceiling)
top       = what is actually being used   (observation)
```

Three different numbers. A node can be 100% *requested* and 5% *used*: the
scheduler refuses new Pods while `kubectl top` shows an idle machine. That gap
is the single most common source of "there is plenty of room, why is it
Pending?".

`kubectl top` reads the metrics API, served by metrics-server. It is a
sampling pipeline, not an instant reading: expect roughly a minute after a Pod
starts before numbers appear.

## Objects involved

- `metrics-server`: a Deployment in `kube-system` serving `metrics.k8s.io`.
- `PodMetrics` / `NodeMetrics`: read-only, never created by hand.
- Node `status.allocatable`: what the scheduler is allowed to hand out.
- Container `resources.requests` / `resources.limits`: the claim and the cap.

## Commands worth knowing

```bash
kubectl top nodes
kubectl top pods -A --sort-by=cpu
kubectl top pods -n NAMESPACE --containers
kubectl describe node NODE            # Allocatable and Allocated resources
kubectl get pod POD -o jsonpath='{.spec.containers[0].resources}'
kubectl -n kube-system get deployment metrics-server
```

## Diagnostic workflow

```text
Reported problem: slow, evicted, throttled or Pending
  ↓ kubectl top nodes
Is real usage high?
  yes → find the consumer: kubectl top pods -A --sort-by=cpu
  no  ↓
kubectl describe node → compare Allocated requests against Allocatable
Are requests near 100% while usage is low?
  yes → the workloads over-request; right-size requests
  no  ↓
Is a single Pod's request larger than any node's allocatable?
  yes → it can never schedule; reduce the request
  no  → look past resources: taints, selectors, affinity, volumes
```

## Common CKA failure modes

- Reading `kubectl top` and concluding there is room, when the scheduler cares
  about requests, not usage.
- A request larger than any node's allocatable: permanently Pending, and no
  amount of free memory changes it.
- `kubectl top` failing with "Metrics API not available" and being mistaken for
  a broken workload rather than a broken metrics-server.
- Confusing a throttled container (CPU limit) with a slow one.
- Forgetting that `top` needs about a minute of history after a Pod starts.

## 5-minute walkthrough

Run `kubectl top nodes`, then `kubectl describe node worker1` and find the
`Allocated resources` block. Compare the percentages: the first is measured,
the second is reserved. Then run `kubectl top pods -A --sort-by=cpu` and pick
out the busiest Pod in the cluster.

## Labs

```bash
dojo start observability-top
dojo start observability-oversized-requests
```
