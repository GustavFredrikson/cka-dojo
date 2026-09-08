# Pod admission

## Mental model

```text
kubectl create Pod
      ↓
  admission           ← LimitRange fills in missing requests/limits
      ↓
  admission           ← ResourceQuota checks the namespace total, may reject
      ↓
   stored in etcd
      ↓
   scheduler
```

Pod Security Admission sits in the same place, and answers a different
question — not "is there room" but "is this Pod allowed to be this
privileged":

```text
kubectl create Pod
      ↓
  admission           ← Pod Security Admission checks the namespace's
      ↓                 pod-security.kubernetes.io/enforce standard
  admission           ← LimitRange fills in missing requests/limits
      ↓
  admission           ← ResourceQuota checks the namespace total
      ↓
   stored in etcd
```

All three act *before* anything is scheduled, and all three are namespaced.
This is why a quota or policy failure never produces a Pending Pod: the Pod is
never created at all. The controller that wanted it keeps retrying, and the error is on the
ReplicaSet, not on a Pod you can describe.

LimitRange writes; ResourceQuota rejects. Together they make a namespace safe
for people who forget to declare resources — the LimitRange gives them
defaults, so the quota has something to count.

## Objects involved

- `ResourceQuota`: hard caps per namespace — object counts (`pods`,
  `services`, `persistentvolumeclaims`) and compute totals (`requests.cpu`,
  `limits.memory`).
- `LimitRange`: per-container `default` (limits), `defaultRequest`, plus `min`
  and `max` bounds enforced at admission.
- `ReplicaSet`: where the rejection is reported when a Deployment is blocked.
- Namespace labels `pod-security.kubernetes.io/enforce`, `/audit` and `/warn`:
  Pod Security Admission is configured entirely by labels, with no object of
  its own. Each takes a standard — `privileged`, `baseline` or `restricted`.

## Commands worth knowing

```bash
kubectl -n NAMESPACE get resourcequota
kubectl -n NAMESPACE describe resourcequota NAME     # Used vs Hard
kubectl -n NAMESPACE get limitrange -o yaml
kubectl -n NAMESPACE describe replicaset RS_NAME     # rejection events
kubectl -n NAMESPACE get events --sort-by=.metadata.creationTimestamp
kubectl create quota NAME --hard=pods=4,requests.cpu=1 -n NAMESPACE
kubectl get ns NAMESPACE --show-labels
kubectl label ns NAMESPACE pod-security.kubernetes.io/enforce=restricted
```

## Diagnostic workflow

```text
A Deployment reports 0 available replicas
  ↓ kubectl get pods -n NAMESPACE
Are there any Pods at all?
  yes → normal Pod troubleshooting (Pending, CrashLoop, probes)
  no  ↓ the Pods were never created; describe the ReplicaSet
Does the event say "exceeded quota"?
  the namespace total is full → raise the quota or shrink the workload
Does it say "must specify requests.cpu" (or similar)?
  the quota constrains a resource the Pod does not declare
    → declare requests on the workload, or add a LimitRange with defaults
Does it say 'violates PodSecurity "restricted:latest"'?
  the namespace enforces a Pod Security Standard the Pod does not meet
    → the message names every rule broken and the field that fixes it;
      set those fields, do not relax the namespace label
```

## Common CKA failure modes

- Looking for Pending Pods when no Pod object was ever created.
- A quota that constrains `requests.cpu` in a namespace whose workloads
  declare nothing, so every Pod is rejected until defaults exist.
- Editing a LimitRange and expecting already-running Pods to change: defaults
  are applied at creation, never retroactively.
- A `limits` value below the matching `requests` value; the API server rejects
  the Pod outright.
- Deleting the quota, or relaxing `enforce` to `baseline`/`privileged`, to make
  the symptom disappear rather than fixing the workload. Both remove the
  control instead of meeting it.
- Applying a Deployment into a `restricted` namespace, seeing it accepted, and
  concluding the policy passed. A Deployment is not a Pod; `enforce` only ever
  looks at Pods, so the rejection surfaces later on the ReplicaSet.
- Putting `allowPrivilegeEscalation` or `capabilities` at Pod level. They are
  container-only fields; `runAsNonRoot` and `seccompProfile` may sit at either.
- Setting `runAsNonRoot: true` on an image that runs as uid 0 without also
  setting `runAsUser`. The Pod is admitted and then fails to start with
  `CreateContainerConfigError`.

## 5-minute walkthrough

Create a namespace, add a quota with `kubectl create quota demo --hard=pods=1`,
then create two Pods. The second one fails, and
`kubectl describe replicaset` — not `describe pod` — tells you why. Then add a
LimitRange and create a Pod with no `resources` block; `kubectl get pod -o yaml`
shows fields you never typed.

## Labs

```bash
dojo start quota-build
dojo start limitrange-build
dojo start quota-blocked
dojo start psa-restricted
```
