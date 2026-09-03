# Cap what a namespace may consume

## Working it out

A ResourceQuota is a flat list of `hard` entries. Two families matter:

- object counts: `pods`, `services`, `persistentvolumeclaims`, …
- compute totals: `requests.cpu`, `requests.memory`, `limits.cpu`,
  `limits.memory`

The compute entries are *sums across the namespace*, not per-Pod ceilings. A
quota of `requests.cpu: 1` with two replicas at `100m` each leaves `800m`
unused.

The catch that bites people: once a quota names `requests.cpu`, every Pod in
that namespace must declare a CPU request or it is rejected outright. That is
why this task asks you to create the workload *with* resources — you will meet
the failure mode in `quota-blocked`.

## Fixing it

Imperatively:

```bash
kubectl create quota team-quota -n dojo-quota-build \
  --hard=pods=4,requests.cpu=1,requests.memory=1Gi,limits.cpu=2,limits.memory=2Gi
```

Or declaratively:

```yaml
apiVersion: v1
kind: ResourceQuota
metadata: {name: team-quota, namespace: dojo-quota-build}
spec:
  hard:
    pods: "4"
    requests.cpu: "1"
    requests.memory: 1Gi
    limits.cpu: "2"
    limits.memory: 2Gi
```

Then the workload. `kubectl create deployment` cannot set resources, so
generate and edit, or create and then `set resources`:

```bash
kubectl -n dojo-quota-build create deployment web --image=nginx:1.27-alpine --replicas=2
kubectl -n dojo-quota-build set resources deployment/web \
  --requests=cpu=100m,memory=64Mi --limits=cpu=200m,memory=128Mi
```

Order matters less than you would think here, but if you create the Deployment
before the quota, the Pods are admitted without scrutiny and the quota simply
starts counting them afterwards.

## Checking before you grade

```bash
kubectl -n dojo-quota-build describe resourcequota team-quota
```

The `Used` column should read `2` pods, `200m` requests.cpu and `128Mi`
requests.memory. If `Used` shows `0` pods while the Deployment reports two
replicas, you are looking at a different namespace.
