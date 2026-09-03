# No replica will schedule anywhere

## Working it out

Zero replicas placed — not two of three — says the Pod template is asking for
somewhere that does not exist, rather than for one slot too many.

```bash
kubectl -n dojo-scheduling-spread-pending get pods -o wide
kubectl -n dojo-scheduling-spread-pending describe pod -l app=checkout | \
  sed -n '/Events/,$p'
```

The `FailedScheduling` event accounts for every node:

```text
0/3 nodes are available: 1 node(s) had untolerated taint(s),
2 node(s) didn't match pod anti-affinity rules.
```

The counts add up to the whole cluster, so nothing is left over. `cp1` is out
on the taint — confirm which node that is rather than assuming:

```bash
kubectl get nodes -o custom-columns=NAME:.metadata.name,TAINTS:.spec.taints
```

That leaves two workers, and both rejected the Pod on anti-affinity. Read the
rule and then find what it is avoiding:

```bash
kubectl -n dojo-scheduling-spread-pending get deploy checkout \
  -o jsonpath='{.spec.template.spec.affinity}'; echo
kubectl -n dojo-scheduling-spread-pending get pods -o wide --show-labels
```

checkout refuses, as a hard requirement, to occupy any node already running a
`tier=cache` Pod. The `cache` Deployment keeps exactly one replica on each
worker. Every schedulable node is therefore disqualified, and no amount of
free CPU changes that — `kubectl top nodes` will show an almost idle cluster.

The requirement itself is reasonable; stating it as an absolute is what breaks.

## Fixing it

Grading accepts either route, and neither may disturb `cache`.

**1. Demote the rule to a preference.**

```yaml
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
            - weight: 100
              podAffinityTerm:
                labelSelector:
                  matchLabels: {tier: cache}
                topologyKey: kubernetes.io/hostname
```

Note the shape change: the preferred form wraps the term in `podAffinityTerm`
and adds a `weight`. Reusing the required form's shape verbatim is the usual
mistake, and the API server will reject it.

**2. Express the intent as topology spread instead.**

```yaml
      topologySpreadConstraints:
        - maxSkew: 1
          topologyKey: kubernetes.io/hostname
          whenUnsatisfiable: ScheduleAnyway
          labelSelector:
            matchLabels: {app: checkout}
```

This drops the cache avoidance and asks for an even spread across nodes
instead. `ScheduleAnyway` is what makes it a preference; `DoNotSchedule` would
reproduce the original deadlock in a new costume.

Apply with `kubectl -n dojo-scheduling-spread-pending edit deployment
checkout`, or by editing a saved manifest and re-applying. Because every
checkout Pod is currently Pending and holds no node, the corrected template
rolls out immediately.

**What not to do.** Deleting the `affinity` block outright, scaling checkout
down, or scaling `cache` down all clear the symptom while discarding the
requirement. Grading checks that a placement preference survives and that
`cache` still has both replicas.

## A trap worth knowing

This exercise points the repulsion at *another* workload on purpose. Had
checkout repelled its own Pods, the repair would have been far nastier:
anti-affinity is **symmetric**, so healthy old Pods keep repelling the
corrected ones. The scheduler cannot place a new Pod until an old one retires,
while `maxUnavailable` can forbid retiring one — and a genuinely correct fix
deadlocks. When that happens on a real cluster, deleting a stale Pod by hand
is what breaks the tie.

## Checking before you grade

```bash
kubectl -n dojo-scheduling-spread-pending get deploy
kubectl -n dojo-scheduling-spread-pending get pods -o wide
```

`checkout` at `3/3` and `cache` still at `2/2`, with checkout's Pods sharing
the workers alongside the cache Pods.
