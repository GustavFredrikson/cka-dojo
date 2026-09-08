# Workload primitives

## Mental model

A Deployment is one answer to "how many of these should run, and where". It is
not the only one, and the others encode different assumptions about replicas.

```text
Is replacing any one Pod free?
  yes → is the count yours to choose?
          yes → Deployment
          no, it is "one per node" → DaemonSet
  no, a Pod owns something (a disk, an ordinal, a peer address)
      → StatefulSet

Is running the goal, or is finishing?
  finishing → Job          (once)
            → CronJob      (a Job factory with a clock)
```

The question is never "which one have I seen before". It is **what does this
workload assume about its own replicas**.

Two derived counts to keep straight, because neither is a number you typed:

```text
Deployment  spec.replicas = 3        ← you chose 3
DaemonSet   status.desiredNumberScheduled = 2
                                     ← derived: nodes it *could* land on
```

A DaemonSet on this cluster reports 2, not 3, until it tolerates the
control-plane taint. Nothing reports an error; the DaemonSet considers itself
complete.

## Objects involved

- `DaemonSet`: no `replicas`. Desired count is node count minus nodes whose
  taints the Pod does not tolerate.
- `Job`: `completions` (how many Pods must succeed), `parallelism` (how many
  at once), `backoffLimit` (how many failed Pods before giving up).
- `CronJob`: `schedule`, `concurrencyPolicy`, `startingDeadlineSeconds`,
  `suspend`, and the history limits. Contains a `jobTemplate`, which contains
  a Pod template — four `spec` levels deep.
- `StatefulSet`: `serviceName` (the governing headless Service),
  `volumeClaimTemplates` (one PVC per Pod, named `<template>-<pod>`),
  ordered creation and reverse-ordered deletion.
- `Service` with `clusterIP: None`: what makes per-Pod DNS names exist.
- `initContainers`: run to completion, in order, before any app container. One
  with `restartPolicy: Always` is a **sidecar** — started before the app,
  kept running alongside it, shut down after it.
- `PriorityClass`: a name mapped to a number. `value` is **immutable**.
  `preemptionPolicy: Never` means "queue ahead of others, but evict nobody".

## Commands worth knowing

```bash
kubectl -n NAMESPACE get daemonset                 # DESIRED vs READY
kubectl create job NAME --image=IMG -- CMD
kubectl create job NAME --from=cronjob/CRONJOB     # run a CronJob now
kubectl create cronjob NAME --image=IMG --schedule='*/5 * * * *' -- CMD
kubectl patch cronjob NAME -p '{"spec":{"suspend":true}}'
kubectl logs POD -c INIT_CONTAINER                 # -c is not optional
kubectl explain cronjob.spec.jobTemplate.spec.template.spec
kubectl get priorityclass
kubectl replace --force -f pc.yaml                 # for immutable fields
kubectl -n NAMESPACE get pods -o wide -w
```

`--dry-run=client -o yaml` works on `create job` and `create cronjob`. There
is no generator for a DaemonSet or a StatefulSet: generate a Deployment and
edit `kind`, or write them out.

## Diagnostic workflow

```text
A DaemonSet reports fewer Pods than there are nodes
  ↓ compare status.desiredNumberScheduled with node count
Are they equal?
  yes → the DaemonSet thinks it is complete; the gap is taints
        → kubectl describe node NODE | grep -i taints, then tolerate it
  no  → normal Pod troubleshooting on the missing node

A Pod sits in Init:0/1
  ↓ it is waiting, not failing
kubectl logs POD -c <init-container>        ← not `kubectl logs POD`
  → whatever it is blocked on is the actual missing object

A Job never completes
  ↓ kubectl describe job NAME
Does it report BackoffLimitExceeded?
  yes → read a failed Pod's logs; restartPolicy: Never keeps them around
  no  → completions vs parallelism: is it just slow, one Pod at a time?

A high-priority Pod stays Pending
  ↓ kubectl describe pod POD | tail -20
Does an event say Insufficient <resource> and nothing about preemption?
  → check the numbers: is its class actually above the incumbents'?
  → check preemptionPolicy: Never cannot evict anyone
```

## Common CKA failure modes

- A DaemonSet that reports `2/2` Ready and is missing from the control-plane
  node entirely: no toleration, and no error anywhere.
- `kubectl logs` on a Pod in `Init:0/1`, reading the error about the app
  container and concluding the app is broken.
- `restartPolicy: Always` in a Job template. The API rejects it — a Pod that
  always restarts can never complete.
- Counting `backoffLimit` as restarts per Pod. It is total failed Pods, with
  an exponentially growing delay.
- Losing the CronJob question to indentation: `jobTemplate.spec.template.spec`
  is four levels of `spec`.
- A StatefulSet whose `serviceName` names no headless Service. The Pods run;
  the per-Pod DNS names silently do not exist.
- Expecting `data-db-0` to disappear with the Pod, or with the StatefulSet.
  Neither deletes it. Cleaning up a StatefulSet is two commands.
- Putting a sidecar in `containers:` instead of making it an init container
  with `restartPolicy: Always`. It ships the same logs and loses every
  ordering guarantee — and in a Job, a plain sidecar container never exits, so
  the Job never completes.
- A sidecar that does not mount the volume it is meant to read. Containers in
  a Pod share a network namespace but not filesystems, only the volumes each
  one asks for.
- `tail -f` where `tail -F` was needed. Lower-case follows the file
  descriptor and goes silent the moment the log rotates, which is exactly the
  case the sidecar was added for.
- Trying to `patch` a PriorityClass's `value`. It is immutable — delete and
  recreate, or `kubectl replace --force`.
- Recreating a PriorityClass and expecting running Pods to change tier.
  Priority is copied into `pod.spec.priority` at admission.

## 5-minute walkthrough

Create a DaemonSet with no tolerations and run
`kubectl get daemonset -o wide` beside `kubectl get nodes`: two against three,
with nothing marked wrong. Add the control-plane toleration and watch the
desired count change from 2 to 3 — a number you never set.

Then create a Job with `completions: 3, parallelism: 2` and watch
`kubectl get pods -w`: two start, and the third only appears as one finishes.

## Labs

```bash
dojo start daemonset-build
dojo start jobs-build
dojo start statefulset-build
dojo start initcontainer-repair
dojo start sidecar-build
dojo start priority-preemption
```
