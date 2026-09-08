# Worked solution

## Why exec fails

```bash
kubectl -n dojo-debug-ephemeral exec -it vault -- sh
```

```
OCI runtime exec failed: exec failed: unable to start container process:
exec: "sh": executable file not found in $PATH: unknown
```

`kubectl exec` runs a binary *from the container's own image*. A distroless
image has no shell to run, so there is nothing to fall back to -- and this is
increasingly normal: smaller attack surface, no shell for an attacker either.
Which is exactly the problem ephemeral containers exist to solve.

## Attaching a debug container

```bash
kubectl -n dojo-debug-ephemeral debug -it vault \
  --image=busybox:1.36 \
  --container=sleuth \
  --target=vault
```

Three flags, three different jobs, and mixing them up is the usual way this
goes wrong:

- `--image` is the image *you* want tooling from.
- `--container` names the ephemeral container. Without it kubectl invents a
  name like `debugger-8f2kx`, which is fine interactively and no good when
  something has to refer to it later.
- `--target` is the container whose **process namespace** you join. Without
  it you get a container in the same Pod -- same network, same volumes it
  declares -- but *not* the same PID namespace, and none of what follows works.

This is a write to the Pod's `ephemeralContainers` subresource, and it is
additive only: you cannot remove an ephemeral container, and you cannot change
one. It disappears when the Pod does.

```bash
kubectl -n dojo-debug-ephemeral get pod vault \
  -o jsonpath='{.spec.ephemeralContainers[*].name}'
```

## Reading the file

Inside `sleuth`, `/run/vault` is not mounted -- an ephemeral container does not
inherit the target's volume mounts, and there is no flag that makes it. What
`--target` gives you is the shared process namespace, and from there the
target's whole filesystem via `/proc`:

```
/ # ps -o pid,user,comm
PID   USER     COMMAND
    1 65535    pause
   13 root     sleep
/ # cat /proc/1/root/run/vault/token
cat: can't open '/proc/1/root/run/vault/token': Permission denied
```

`--target` worked -- PID 1 is the target container's process, because a
container's main process is PID 1 in its own namespace and you are now sharing
that namespace. But reading through `/proc/1/root` is refused, and the reason
is the useful part of this exercise.

Look at the two users. The debug container is `root`. The target runs as
**65535**, because distroless images are usually built to run unprivileged and
this one declares a non-root `USER`. Reading `/proc/<pid>/root` for a process
you do not own needs `CAP_SYS_PTRACE`, and the default capability set a
container gets does not include it. So: right namespace, wrong privileges.

That is what `kubectl debug`'s **profiles** are for:

```bash
kubectl -n dojo-debug-ephemeral debug vault \
  --image=busybox:1.36 --container=sleuth --target=vault \
  --profile=sysadmin -- sleep 3600
```

```
/ # cat /proc/1/root/run/vault/token
7c2f9a5e1b
```

`--profile` sets the ephemeral container's security context for you:

- `legacy` -- the historical default, and what you get if you say nothing.
  No added privileges. Fine for network probing, useless for reading another
  container's files.
- `general` -- the sane default for most debugging.
- `baseline` / `restricted` -- for clusters with Pod Security admission that
  would reject anything more.
- `sysadmin` -- privileged. What you need here, and what you should reach for
  last rather than first.
- `netadmin` -- adds `NET_ADMIN`/`NET_RAW`, for `tcpdump` and `ip`.

There is one trap worth knowing before you type anything. **The
`ephemeralContainers` subresource is append-only.** You cannot remove an
ephemeral container, rename it, or change its profile; it lives until the Pod
does. So if your first attempt used the wrong profile, that name is spent --
attach another one under a different name rather than trying to fix it. Add
`--profile` from the start, and give the container a name you chose (`sleuth`
here) rather than letting kubectl invent `debugger-8f2kx`.

`/proc/<pid>/root` is the single most useful thing to know about debugging
containers you cannot get a shell in. `/proc/1/environ` and `/proc/1/cwd` work
the same way, which is how you check what a container was actually started
with rather than what its spec says.

If `ps` shows no PID 1 other than your own shell, `--target` did not take
effect -- check the flag, and check that `kubectl get pod -o yaml` records
`targetContainerName`.

## Storing it

```bash
kubectl -n dojo-debug-ephemeral create secret generic recovered \
  --from-literal=token=7c2f9a5e1b
```

## The other route, and why it is not this one

```bash
kubectl -n dojo-debug-ephemeral debug vault --copy-to=vault-debug \
  --set-image=vault=busybox:1.36 -- sleep 3600
kubectl -n dojo-debug-ephemeral exec -it vault-debug -- cat /run/vault/token
```

`--copy-to` creates a *new* Pod from the original's spec, so `--set-image` can
swap the unshelled image for one you can work in, and the copy keeps the
volumes. It is the right tool when the original is crash-looping and there is
no running process to attach to at all.

It is the wrong tool here, for the reason the task rules it out: it does not
touch the running Pod. Anything you learn is from a copy, whose state may
differ from the one actually misbehaving -- and if the file had been written
from something that only happened once, the copy would not have it.

## Checking before you grade

```bash
kubectl -n dojo-debug-ephemeral get pod vault
kubectl -n dojo-debug-ephemeral get pod vault \
  -o jsonpath='{.spec.ephemeralContainers[*].targetContainerName}'
kubectl -n dojo-debug-ephemeral get secret recovered -o jsonpath='{.data.token}' | base64 -d
```

`vault` still `Running` with `0` restarts, `sleuth` targeting `vault`, and the
token decoding to what you read.

One honest note on this exercise: the token is base64 in the Pod spec, so it
*can* be recovered by reading the manifest instead of the container. Grading
requires an ephemeral container attached to `vault` so that the shortcut
cannot pass on its own -- but on a real cluster the file would be genuinely
unreachable any other way, and the technique above is the only one that gets
you there.
