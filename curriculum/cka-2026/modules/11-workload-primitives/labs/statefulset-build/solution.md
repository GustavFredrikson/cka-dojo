# Worked solution

## Working it out

There is no generator for a StatefulSet. Write it, and get the three couplings
right -- they are what the exercise is testing, and each one fails quietly.

**The Service must be headless, and the StatefulSet must name it.**
`clusterIP: None` is what makes it headless: DNS returns the Pod addresses
instead of one virtual IP. `spec.serviceName: db` is what tells the
StatefulSet which Service governs it. Omit either and the Pods still start --
they just have no per-Pod DNS names, so the reason you chose a StatefulSet is
silently gone.

**The claim template's name decides the claim's name.** A template called
`data` produces `data-db-0` and `data-db-1`. You do not create those; the
controller does, one per Pod, and they are what makes storage follow an
ordinal.

**The volumeMount must reference the template by name.** The template is not
mounted automatically.

## Writing it

```yaml
apiVersion: v1
kind: Service
metadata:
  name: db
  namespace: dojo-statefulset-build
spec:
  clusterIP: None
  selector: {app: db}
  ports:
    - port: 5432
      name: postgres
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: db
  namespace: dojo-statefulset-build
spec:
  serviceName: db
  replicas: 2
  selector:
    matchLabels: {app: db}
  template:
    metadata:
      labels: {app: db}
    spec:
      containers:
        - name: db
          image: busybox:1.36
          command: ["sleep", "3600"]
          volumeMounts:
            - name: data
              mountPath: /data
  volumeClaimTemplates:
    - metadata:
        name: data
      spec:
        accessModes: [ReadWriteOnce]
        storageClassName: dojo-statefulset
        resources:
          requests:
            storage: 1Gi
```

Watch it come up and you can see the ordering guarantee at work:

```bash
kubectl -n dojo-statefulset-build get pods -w
```

`db-0` reaches Ready before `db-1` is created. That is deliberate, and it is
why a StatefulSet with a Pod that never becomes Ready stops dead at that
ordinal instead of carrying on. If you are stuck at `db-0` Pending, look at
the claim -- `WaitForFirstConsumer` means the claim binds after scheduling, so
a Pending Pod and a Pending claim are the same problem, not two.

## The DNS part

```bash
kubectl -n dojo-statefulset-build exec db-0 -- \
  nslookup db-1.db.dojo-statefulset-build.svc.cluster.local
```

The name has five parts and every one comes from something you wrote:
`<pod>.<service>.<namespace>.svc.cluster.local`. This is what a database peer
list is built from, and it is the reason ordinals have to be stable -- a
replacement `db-1` gets the same name and the same volume.

## Checking before you grade

```bash
kubectl -n dojo-statefulset-build get statefulset,pods,pvc
kubectl -n dojo-statefulset-build get svc db -o jsonpath='{.spec.clusterIP}'
```

Two Ready Pods, `data-db-0` and `data-db-1` both `Bound`, and `None` for the
cluster IP.

One thing worth knowing before you delete anything: those claims outlive both
the Pods and the StatefulSet. Scaling to zero and back reattaches the same
volumes; deleting the StatefulSet leaves the claims behind. That is the
guarantee, not a leak -- but it means cleaning up a StatefulSet is two
commands, not one.
