# A Deployment that never gets a node

## Working it out

Nothing is crash-looping and no image is wrong: the Pods are `Pending`, which
means the scheduler never bound them. The scheduler always says why.

```bash
kubectl -n dojo-observability-oversized-requests get pods
kubectl -n dojo-observability-oversized-requests describe pod -l app=reporting
```

The Events end with a `FailedScheduling` line naming every node and the reason
`Insufficient cpu`.

The trap is that the cluster really is idle:

```bash
kubectl top nodes
```

Usage is near zero on all three nodes, which tempts you to look for a taint or
a selector. It is neither. The scheduler compares the container's *request*
against the node's *allocatable*, and never looks at measured usage:

```bash
kubectl -n dojo-observability-oversized-requests get deploy reporting \
  -o jsonpath='{.spec.template.spec.containers[0].resources}'; echo
kubectl describe node worker1 | sed -n '/Allocatable/,/System Info/p'
```

Each replica asks for `3` CPU. Every worker is a 2-CPU VM with roughly `1900m`
allocatable. No node can ever satisfy that request, so the Pods stay Pending
forever — this is not a transient capacity problem.

## Fixing it

Any route that leaves the Deployment with a request the cluster can satisfy is
correct. Three equivalent ones:

```bash
# 1. set resources
kubectl -n dojo-observability-oversized-requests set resources \
  deployment/reporting --requests=cpu=100m,memory=64Mi --limits=cpu=500m,memory=128Mi

# 2. edit
kubectl -n dojo-observability-oversized-requests edit deployment reporting

# 3. patch
kubectl -n dojo-observability-oversized-requests patch deployment reporting --type=json \
  -p '[{"op":"replace","path":"/spec/template/spec/containers/0/resources/requests/cpu","value":"100m"},
       {"op":"replace","path":"/spec/template/spec/containers/0/resources/limits/cpu","value":"500m"}]'
```

Keep a request in place rather than deleting the `resources` block. Removing
it also schedules the Pods, but it hands the scheduler no information at all
and is what produces overcommitted nodes later.

Note that the limit must come down with the request: a limit below the request
is rejected by the API server.

## Checking before you grade

```bash
kubectl -n dojo-observability-oversized-requests get deploy reporting
kubectl -n dojo-observability-oversized-requests get pods -o wide
```

Both replicas should be `Available`, and the two Pods should be spread across
worker nodes.
