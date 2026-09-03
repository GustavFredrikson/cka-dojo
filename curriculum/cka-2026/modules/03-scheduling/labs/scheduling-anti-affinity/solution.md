# Keep two replicas off the same node

## Working it out

"Refuse" rather than "prefer" is the whole specification. Anti-affinity comes
in two strengths and they behave completely differently when the cluster runs
out of room:

- `requiredDuringSchedulingIgnoredDuringExecution` — a hard predicate. If no
  node satisfies it, the Pod stays Pending forever.
- `preferredDuringSchedulingIgnoredDuringExecution` — a scoring nudge. The
  scheduler spreads if it can and packs if it cannot.

The task asks for the hard form.

The second thing to get right is `topologyKey`. It names the node label whose
value defines "the same place". `kubernetes.io/hostname` is per-node, which is
what "not on the same node" means. Using a zone label instead would allow two
Pods on one node inside the same zone.

Note that the `labelSelector` inside the anti-affinity rule selects the *other
Pods to stay away from* — here, the Deployment's own Pods. It is not the
Deployment's `spec.selector`, even though both end up saying `app=web`.

## Fixing it

Generate the skeleton, then add the block by hand — `kubectl create` cannot
express affinity:

```bash
kubectl -n dojo-scheduling-anti-affinity create deployment web \
  --image=nginx:1.27-alpine --replicas=2 --dry-run=client -o yaml > web.yaml
```

```yaml
    spec:
      affinity:
        podAntiAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            - labelSelector:
                matchLabels: {app: web}
              topologyKey: kubernetes.io/hostname
      containers:
        - name: nginx
          image: nginx:1.27-alpine
```

```bash
kubectl apply -f web.yaml
```

`kubectl explain deployment.spec.template.spec.affinity.podAntiAffinity` is
the fastest way to recall the field names under exam pressure — the nesting is
deep and easy to get wrong.

## Checking before you grade

```bash
kubectl -n dojo-scheduling-anti-affinity get pods -o wide
```

Two Pods, both Running, with two different values in the `NODE` column. Only
`worker1` and `worker2` are schedulable — `cp1` carries the control-plane
taint — so a third replica would stay Pending. That is the subject of
`scheduling-spread-pending`.
