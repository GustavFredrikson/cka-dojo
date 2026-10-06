# Worked solution

## Working it out

Start at the claim and follow what it is waiting for:

```bash
kubectl -n dojo-csi-provisioner-restore describe pvc reports
```

The events say `waiting for a volume to be created, either by external
provisioner "rancher.io/local-path" or manually by system administrator`. That
sentence contains the whole diagnosis if you read it as a question: *is* that
external provisioner running?

The StorageClass names it:

```bash
kubectl get storageclass local-path -o jsonpath='{.provisioner}{"\n"}'
```

`rancher.io/local-path` is not a built-in. Nothing in the API server or the
controller-manager creates these volumes — a separate Deployment watches for
Pending claims in that class and provisions them. That is the extension point
this lab is about: storage in Kubernetes is delegated, and when it stops, the
symptom appears on the claim while the cause is a workload somewhere else.

```bash
kubectl -n local-path-storage get deploy,pods
```

From there the two variants diverge:

- **nothing is there.** The Deployment is scaled to 0, no Pod at all.
- **something is there and cannot start.** The Deployment is scaled to 1 and
  its Pod is `ImagePullBackOff` or `ErrImagePull`.
  `kubectl -n local-path-storage describe pod ...` names an image tag that does
  not exist.

Note also what is *not* wrong: the class is `WaitForFirstConsumer`, so a
Pending claim with no consumer would be normal. There is a consumer here — the
`archiver` Pod — so waiting is not the explanation.

## Fixing it

**Scaled down:**

```bash
kubectl -n local-path-storage scale deploy/local-path-provisioner --replicas=1
kubectl -n local-path-storage rollout status deploy/local-path-provisioner
```

**Bad image:** put the working tag back. The ReplicaSet history still has it:

```bash
kubectl -n local-path-storage rollout history deploy/local-path-provisioner
kubectl -n local-path-storage rollout undo deploy/local-path-provisioner
```

or set it directly:

```bash
kubectl -n local-path-storage set image deploy/local-path-provisioner \
  local-path-provisioner=docker.io/rancher/local-path-provisioner:v0.0.37
```

Either way the provisioner comes back, notices the Pending claim, creates a
PersistentVolume, and the claim binds within a few seconds. The `archiver` Pod
then schedules on its own.

## What is not a fix

Creating the PersistentVolume by hand:

```bash
# Do not do this here.
kubectl apply -f my-hand-written-pv.yaml
```

The claim would bind and the Pod would start — and dynamic provisioning would
still be broken for everything else. Grading reads the
`pv.kubernetes.io/provisioned-by` annotation on the bound volume, which only the
provisioner sets, so a hand-written PV fails.

That distinction is the point. On the exam, "the claim is Bound" and "storage
works" are not the same statement.

## Checking before you grade

```bash
kubectl -n local-path-storage get deploy local-path-provisioner
kubectl -n dojo-csi-provisioner-restore get pvc,pod
kubectl get pv -o custom-columns=\
NAME:.metadata.name,BY:.metadata.annotations.'pv\.kubernetes\.io/provisioned-by'
kubectl -n dojo-csi-provisioner-restore exec archiver -- cat /data/marker
```

The last command should print `ok` — the Pod wrote to the volume it was given.
