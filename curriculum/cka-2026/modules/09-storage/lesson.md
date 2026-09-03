# Storage

## Mental model

```text
Pod → PVC → matching PV → storage
        ↑
  StorageClass policy
```

A claim describes requested capacity, access modes and class. Binding needs a
compatible available volume, and there are two ways to get one.

*Static*: an administrator creates the PersistentVolume ahead of time, and the
claim has to match something that already exists. Most exercises here work
this way, because it keeps the matching rules visible.

*Dynamic*: the claim names a StorageClass, and its provisioner creates a
volume to fit. This cluster runs `rancher.io/local-path`, so
`storage-dynamic` practises the real thing rather than describing it.

`volumeBindingMode` decides *when* that happens. `Immediate` provisions as
soon as the claim exists; `WaitForFirstConsumer` holds off until a Pod is
scheduled, so node-local storage lands on the node that will actually use it.
A claim sitting Pending with `waiting for first consumer` is working
correctly.

## Objects involved

- `PersistentVolume`: cluster-scoped capacity and backing storage.
- `PersistentVolumeClaim`: namespaced request for storage.
- `StorageClass`: provisioning and binding policy.
- Pod volume: references a claim and mounts it into a container.

## Commands worth knowing

```bash
kubectl get pv,pvc -A
kubectl describe pv PV_NAME
kubectl describe pvc CLAIM -n NAMESPACE
kubectl get storageclass
kubectl get events --sort-by=.metadata.creationTimestamp
kubectl get pod POD -o yaml
```

## Diagnostic workflow

```text
PVC Pending
  ↓ describe claim and inspect Events
Does storageClassName match an available PV or provisioner?
  no → correct class/provisioning expectation
  yes ↓
Does requested capacity fit?
  no → reduce request or provide larger PV
  yes ↓
Do access modes and volumeMode match?
  no → make request and volume compatible
  yes ↓
Check selector, topology and WaitForFirstConsumer
```

## Common CKA failure modes

- PVC and static PV use different `storageClassName` values.
- Requested capacity exceeds the PV.
- Claim asks for `ReadWriteMany` from a `ReadWriteOnce` volume.
- A Pod references the wrong claim name or namespace.
- `WaitForFirstConsumer` is mistaken for broken provisioning.
- Reclaim policy behavior is ignored when a claim is deleted.

## 5-minute walkthrough

Use `kubectl explain pv.spec`, `kubectl explain pvc.spec`, and compare a bound
pair with `kubectl get pv,pvc -o wide`. Focus on class, capacity, access modes,
status phase and the claim's `spec.volumeName`.

## Labs

```bash
dojo start storage-follow
dojo start storage-build
dojo start storage-dynamic
dojo start storage-reclaim
dojo start storage-inspect
dojo start storage-guided-pending
dojo start storage-pending
```
