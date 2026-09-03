# Let the cluster create the volume for you

## Working it out

Dynamic provisioning replaces the human who would otherwise create a
PersistentVolume. A StorageClass names the provisioner to call and the policy
to apply; the claim references the class; the provisioner produces a volume
sized to the claim.

Two fields carry the whole lesson:

- `provisioner` — who creates the volume. Here `rancher.io/local-path`, which
  carves a directory out of the node the consuming Pod lands on. Check what
  the cluster actually has with `kubectl get storageclass`.
- `volumeBindingMode` — *when* to create it. `Immediate` provisions as soon as
  the claim exists. `WaitForFirstConsumer` waits until a Pod is scheduled, and
  that is what the task asks for.

`WaitForFirstConsumer` exists because of node-local storage exactly like this
one. If the volume were created immediately, it would be pinned to some node
chosen before the Pod was scheduled, and the Pod might then be unable to run
there. Deferring means the scheduler picks the node first and the volume
follows.

## Fixing it

The class cannot be edited into shape. Try it and the API server says so:

```bash
kubectl patch storageclass dojo-dynamic -p '{"provisioner":"rancher.io/local-path"}'
# The StorageClass "dojo-dynamic" is invalid: provisioner: Forbidden:
# updates to provisioner are forbidden.
```

`provisioner`, `parameters` and `volumeBindingMode` are immutable once a
StorageClass exists. Changing any of them means delete and recreate — worth
knowing before you waste time in `kubectl edit`:

```bash
kubectl delete storageclass dojo-dynamic
```

```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata: {name: dojo-dynamic}
provisioner: rancher.io/local-path
reclaimPolicy: Delete
volumeBindingMode: WaitForFirstConsumer
```

Note the shape: `provisioner`, `reclaimPolicy` and `volumeBindingMode` are
top-level fields on a StorageClass. There is no `spec`, which trips up almost
everyone the first time.

Then the claim:

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata: {name: scratch, namespace: dojo-storage-dynamic}
spec:
  accessModes: [ReadWriteOnce]
  storageClassName: dojo-dynamic
  resources: {requests: {storage: 1Gi}}
```

Apply just those two and look before going on:

```bash
kubectl -n dojo-storage-dynamic get pvc scratch
kubectl -n dojo-storage-dynamic describe pvc scratch | tail -5
```

The claim is `Pending`, and the reason reads
`waiting for first consumer to be created before binding`. **This is correct
behaviour, not a fault.** Recognising that message is worth more than the rest
of the exercise — it is the one Pending claim you should not try to fix.

Now give it a consumer:

```yaml
apiVersion: v1
kind: Pod
metadata: {name: writer, namespace: dojo-storage-dynamic}
spec:
  containers:
    - name: writer
      image: busybox:1.36
      command: ["sh", "-c", "sleep 3600"]
      volumeMounts:
        - {name: data, mountPath: /data}
  volumes:
    - name: data
      persistentVolumeClaim: {claimName: scratch}
```

The claim binds within seconds of the Pod being scheduled.

## Checking before you grade

```bash
kubectl -n dojo-storage-dynamic get pvc scratch
kubectl get pv
```

The claim is `Bound`, and `kubectl get pv` lists a volume named `pvc-<uuid>`
that you never wrote — its `CLAIM` column points back at
`dojo-storage-dynamic/scratch`. That generated name is the signature of
dynamic provisioning; a statically created volume keeps the name its author
gave it.
