# Solution

## Working it out

```bash
kubectl -n dojo-psa get deployment,replicaset,pods
```

```
deployment.apps/report   0/2
replicaset.apps/report-6c8d99b47   2   0   0
No resources found in dojo-psa namespace.     <- no Pods
```

Two desired, zero created. This is the same shape as a ResourceQuota
rejection and the same lesson: **when there is no Pod, the Pod is not where
the error is.** Something refused the create call, and the thing that made the
call was the ReplicaSet controller:

```bash
kubectl -n dojo-psa describe replicaset report-6c8d99b47 | tail -12
```

```
Warning  FailedCreate  ...  Error creating: pods "report-..." is forbidden:
violates PodSecurity "restricted:latest": allowPrivilegeEscalation != false
(container "report" must set securityContext.allowPrivilegeEscalation=false),
unrestricted capabilities (container "report" must set
securityContext.capabilities.drop=["ALL"]), runAsNonRoot != true (pod or
container "report" must set securityContext.runAsNonRoot=true), seccompProfile
(pod or container "report" must set securityContext.seccompProfile.type to
"RuntimeDefault" or "Localhost")
```

That message is unusually generous — it lists every rule broken, and for each
one, the exact field to set. Read it rather than reciting the standard from
memory.

Where the policy comes from:

```bash
kubectl get ns dojo-psa --show-labels
```

```
pod-security.kubernetes.io/enforce=restricted
pod-security.kubernetes.io/enforce-version=latest
pod-security.kubernetes.io/warn=restricted
```

Pod Security Admission is a built-in admission controller configured entirely
by **namespace labels**. Three modes, and they are independent:

- `enforce` — reject the Pod. The only one that blocks anything.
- `audit` — allow it, record it in the audit log.
- `warn` — allow it, print a warning to the client.

Three standards: `privileged` (no restrictions), `baseline` (blocks known
privilege escalations), `restricted` (the hardened one, in force here).

The `warn` label is why applying this Deployment by hand prints a warning and
still succeeds — the Deployment is not a Pod, so `enforce` never looks at it.
The rejection happens later, to the Pods the ReplicaSet creates. A Deployment
that applies cleanly and produces no Pods is the signature.

## Fixing it

The task says the policy stays. So satisfy it:

```bash
kubectl -n dojo-psa edit deployment report
```

```yaml
    spec:
      securityContext:
        runAsNonRoot: true
        runAsUser: 1000
        seccompProfile:
          type: RuntimeDefault
      containers:
        - name: report
          image: busybox:1.36
          command: ["sleep", "3600"]
          securityContext:
            allowPrivilegeEscalation: false
            capabilities:
              drop: ["ALL"]
```

Which level each field goes at is the part worth getting right:

- `runAsNonRoot` and `seccompProfile` may be set on the **Pod** and inherited
  by every container. Setting them once is less to get wrong.
- `allowPrivilegeEscalation` and `capabilities` are **container** fields only.
  There is nowhere else to put them, and PSA checks every container —
  including init containers and ephemeral containers, which is why adding a
  debug container to a restricted Pod needs `--profile=restricted`.

`runAsNonRoot: true` is an assertion the kubelet verifies at start-up, not
something that makes the container non-root: if the image would run as uid 0
and you only claim otherwise, the Pod is admitted and then fails to start with
`CreateContainerConfigError`. Set `runAsUser` to a non-zero value as well, and
you have said it and made it true.

The rollout is immediate; each corrected Pod is admitted as it is created.

## What would have been the wrong answer

```bash
kubectl label ns dojo-psa pod-security.kubernetes.io/enforce=privileged --overwrite
kubectl label ns dojo-psa pod-security.kubernetes.io/enforce-      # or remove it
```

Both make the symptom vanish in one command, and both delete the control
instead of meeting it. Grading checks the label is still `restricted`, and
also checks the four fields are genuinely present — so exempting the workload
rather than fixing it does not pass either.

## Checking before you grade

```bash
kubectl -n dojo-psa get deployment report
kubectl -n dojo-psa get pods
kubectl get ns dojo-psa -o jsonpath='{.metadata.labels}' ; echo
```

`2/2` available, two Running Pods, and the namespace still labelled
`restricted`.
