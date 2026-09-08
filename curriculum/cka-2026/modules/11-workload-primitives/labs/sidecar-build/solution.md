# Worked solution

## What a sidecar actually is

Since Kubernetes 1.29 a sidecar is a first-class thing, and its definition
looks like a contradiction until you read it twice:

> a sidecar is an **init container** with `restartPolicy: Always`.

Init containers normally run to completion, in order, before the app
containers start. Give one `restartPolicy: Always` and three things change:

- the Pod's startup waits for it to be **started**, not to **finish**;
- it keeps running alongside the app containers for the life of the Pod;
- it is shut down **after** them, so it can still ship the app's last words.

That ordering is the whole reason the feature exists. Before it, a sidecar was
an ordinary container in `containers:` with no ordering guarantee at all — a
log shipper could start after the app and lose the first seconds of output,
and could be killed before the app during termination and lose the last.

It also fixes sidecars in Jobs. A plain sidecar container never exits, so a
Job containing one never completes; a `restartPolicy: Always` init container
is excluded from that calculation.

## Writing it

```bash
kubectl -n dojo-sidecar edit deployment app
```

```yaml
    spec:
      initContainers:
        - name: log-shipper
          image: busybox:1.36
          restartPolicy: Always          # this line is what makes it a sidecar
          command: ["sh", "-c", "tail -n+1 -F /var/log/app/app.log"]
          volumeMounts:
            - name: logs
              mountPath: /var/log/app
      containers:
        - name: app
          ...
```

Two details that decide whether this works:

**It must mount the same volume.** Containers in a Pod share the network
namespace by default but *not* filesystems — only the volumes each one asks
for. The sidecar has to declare the same `logs` volume, or it will tail a path
that does not exist and crash-loop.

**`tail -F`, not `tail -f`.** Capital `-F` re-opens the file if it is rotated
or replaced; lower-case `-f` follows the file descriptor and goes silent the
moment anything rotates the log — which is precisely the situation you added a
log shipper for. `-n+1` starts from the beginning of the file rather than the
last ten lines.

## Why not just add it to `containers:`

It would work, today, for this Pod. An ordinary container in the same Pod with
the same volume mount ships the same lines. What you would lose is every
guarantee above: no ordering on start-up, no ordering on shutdown, and a Job
built the same way would hang forever.

Grading checks `initContainers[log-shipper].restartPolicy == Always` for that
reason — the task is about the shape, not only the log output.

## Checking before you grade

```bash
kubectl -n dojo-sidecar get pods
POD=$(kubectl -n dojo-sidecar get pods -l app=app \
        --field-selector=status.phase=Running -o name | head -1)
kubectl -n dojo-sidecar logs "$POD" -c log-shipper --tail=5
kubectl -n dojo-sidecar get "$POD" \
  -o jsonpath='{.status.initContainerStatuses[*].state}' ; echo
```

Name the Pod rather than reaching for `deployment/app` or `.items[0]`, and
wait for the rollout to finish first. While it is in progress two Pods carry
the `app=app` label, and `kubectl logs deployment/app` picks whichever it
likes — frequently the *old* one, which has no `log-shipper` container:

```
Found 2 pods, using pod/app-8454c74d89-7m4ml
error: container log-shipper is not valid for pod app-8454c74d89-7m4ml
```

That error says your sidecar is missing. It is not; you are looking at the
Pod you replaced.

The Pod reads `2/2` Ready — a running sidecar counts towards the ready
containers, unlike an ordinary init container. The sidecar's state is
`running`, not `terminated`. And its log stream carries `dojo-ledger-entry`
lines while `kubectl logs -c app` still shows nothing, which is the before and
after in one command.
