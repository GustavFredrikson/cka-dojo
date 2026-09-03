# Keep the volume when the claim goes away

## Working it out

The reclaim policy is a property of the *volume*, not of the claim, and it is
read at the moment the claim is deleted. There are two policies that matter in
practice:

- `Delete` — the volume object disappears with the claim. This is the default
  for dynamically provisioned volumes, and it is what destroys data.
- `Retain` — the volume survives and moves to `Released`. The data stays; the
  volume is not offered to another claim until an administrator clears
  `spec.claimRef`.

`Recycle` exists in old material and is deprecated. Do not reach for it.

## Fixing it

The volume already exists with `Delete`, so the quickest correct route is to
change the policy in place:

```bash
kubectl patch pv dojo-storage-reclaim \
  -p '{"spec":{"persistentVolumeReclaimPolicy":"Retain"}}'
```

`kubectl edit pv dojo-storage-reclaim` does the same thing, and so does
deleting the volume and recreating it with the policy set — grading looks at
the resulting state, not the route:

```yaml
apiVersion: v1
kind: PersistentVolume
metadata: {name: dojo-storage-reclaim}
spec:
  capacity: {storage: 2Gi}
  accessModes: [ReadWriteOnce]
  storageClassName: dojo-static-reclaim
  persistentVolumeReclaimPolicy: Retain
  hostPath: {path: /var/lib/dojo/storage-reclaim, type: DirectoryOrCreate}
```

Change the policy *before* you delete the claim. Deleting a claim while the
volume still says `Delete` destroys the volume, and there is nothing to
recover — you would have to recreate it and start over.

Then the claim:

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata: {name: archive, namespace: dojo-storage-reclaim}
spec:
  accessModes: [ReadWriteOnce]
  storageClassName: dojo-static-reclaim
  resources: {requests: {storage: 1Gi}}
```

```bash
kubectl apply -f pvc.yaml
kubectl -n dojo-storage-reclaim get pvc archive     # Bound
kubectl delete pvc archive -n dojo-storage-reclaim
kubectl get pv dojo-storage-reclaim                 # Released, not gone
```

## Checking before you grade

```bash
kubectl get pv dojo-storage-reclaim \
  -o custom-columns=NAME:.metadata.name,POLICY:.spec.persistentVolumeReclaimPolicy,PHASE:.status.phase
kubectl -n dojo-storage-reclaim get pvc
```

You want `Retain` / `Released`, and no claims left in the namespace. A volume
still showing `Bound` means the claim was never deleted; one showing
`Available` means it was never bound in the first place.
