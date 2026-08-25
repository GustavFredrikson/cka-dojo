# Scheduling

## Mental model

The scheduler filters nodes that cannot run a Pod, scores the remaining
candidates, and binds the Pod to one node. A Pod stays Pending when every node
is filtered out.

```text
Pod requirements
  ├─ resource requests
  ├─ nodeSelector / affinity
  └─ tolerations
          ↓ filter
Node properties
  ├─ allocatable resources
  ├─ labels
  └─ taints
          ↓
     spec.nodeName
```

A selector attracts a Pod to matching nodes. A taint repels Pods; a matching
toleration permits scheduling but does not attract the Pod to that node.

## Objects involved

- `Pod`: requests scheduling through selectors, affinity and tolerations.
- `Node`: advertises labels, capacity, conditions and taints.
- `Deployment`: owns a Pod template; scheduling fields belong under
  `spec.template.spec`.
- Events: record why the scheduler rejected each candidate node.

## Commands worth knowing

```bash
kubectl get nodes --show-labels
kubectl describe node worker2
kubectl taint node worker2 dedicated=training:NoSchedule
kubectl label node worker2 tier=special
kubectl get pod web -o wide
kubectl describe pod web
kubectl get events --sort-by=.lastTimestamp
```

For a Pending Pod, `kubectl describe pod` is usually the fastest first move.
Read the `FailedScheduling` event before editing anything.

## Diagnostic workflow

```text
Pod Pending
  ↓ describe pod and read FailedScheduling
Does a node match nodeSelector / required affinity?
  no → compare Pod constraints with node labels
  yes ↓
Does an unmatched NoSchedule taint repel it?
  yes → inspect taints and tolerations
  no  ↓
Do requests fit allocatable resources?
  no → compare requests with node capacity and current use
  yes ↓
Check claims, topology constraints and scheduler health
```

## Common CKA failure modes

- A `nodeSelector` value does not exist on any node.
- A Pod is pinned to a tainted node without a matching toleration.
- A toleration exists, but no selector attracts the Pod to the intended node.
- Scheduling fields are placed on a Deployment rather than its Pod template.
- CPU or memory requests exceed every node's available capacity.
- A node is cordoned (`spec.unschedulable: true`).

## 5-minute walkthrough

Label a worker, create a selected Pod, and inspect the binding:

```bash
kubectl label node worker2 dojo-demo=special
kubectl run selected --image=nginx:1.27-alpine \
  --overrides='{"spec":{"nodeSelector":{"dojo-demo":"special"}}}'
kubectl get pod selected -o wide
kubectl describe pod selected
kubectl delete pod selected
kubectl label node worker2 dojo-demo-
```

## Labs

The path begins with visible scheduler constraints and removes support one
stage at a time:

```bash
dojo start scheduling-follow
dojo start scheduling-build
dojo start scheduling-inspect
dojo start scheduling-guided-toleration
dojo start scheduling-pending
```
