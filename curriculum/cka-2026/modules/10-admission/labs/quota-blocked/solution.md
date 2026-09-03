# A Deployment with no Pods at all

## Working it out

The absence of Pods is the whole clue. Every failure mode that produces a
Pending, CrashLooping or ImagePullBackOff Pod requires a Pod object to exist
first. Here there is none, so the request to create one was refused — that
only happens at admission.

The Deployment does not create Pods. Its ReplicaSet does, and that is where
the refusal is recorded:

```bash
kubectl -n dojo-quota-blocked get deploy,rs
kubectl -n dojo-quota-blocked describe rs -l app=worker
```

The event reads roughly:

```text
Error creating: pods "worker-..." is forbidden: failed quota: team-quota:
must specify limits.cpu for: worker; limits.memory for: worker;
requests.cpu for: worker; requests.memory for: worker
```

`kubectl get events` shows the same thing if you sort it:

```bash
kubectl -n dojo-quota-blocked get events --sort-by=.metadata.creationTimestamp
```

Now the rule behind it:

```bash
kubectl -n dojo-quota-blocked describe resourcequota team-quota
kubectl -n dojo-quota-blocked get deploy worker \
  -o jsonpath='{.spec.template.spec.containers[0].resources}'; echo
```

The quota tracks `requests.cpu`, `requests.memory`, `limits.cpu` and
`limits.memory`. The Pod template declares none of them. A quota can only
count what a Pod states, so it refuses any Pod that leaves a tracked resource
undeclared. This is not "the namespace is full" — `Used` is `0` of everything.

## Fixing it

Two genuinely different routes, and grading accepts both.

**1. Declare the resources on the workload.**

```bash
kubectl -n dojo-quota-blocked set resources deployment/worker \
  --requests=cpu=100m,memory=64Mi --limits=cpu=200m,memory=128Mi
```

**2. Let the namespace supply them.** This is the better answer when you do not
control the manifests:

```yaml
apiVersion: v1
kind: LimitRange
metadata: {name: defaults, namespace: dojo-quota-blocked}
spec:
  limits:
    - type: Container
      defaultRequest: {cpu: 100m, memory: 64Mi}
      default: {cpu: 200m, memory: 128Mi}
```

```bash
kubectl apply -f limitrange.yaml
kubectl -n dojo-quota-blocked rollout restart deployment/worker
```

The restart matters: defaulting happens when a Pod is created, so the
ReplicaSet has to try again after the LimitRange exists.

**What not to do.** `kubectl delete resourcequota team-quota` makes both
replicas appear immediately and is the wrong answer — it removes the
protection instead of satisfying it. Grading checks that the quota survived
and is counting the two new Pods.

## Checking before you grade

```bash
kubectl -n dojo-quota-blocked get deploy worker
kubectl -n dojo-quota-blocked describe resourcequota team-quota
```

`worker` should read `2/2`, and the quota's `Used` column should show `2` pods
with non-zero CPU and memory.
