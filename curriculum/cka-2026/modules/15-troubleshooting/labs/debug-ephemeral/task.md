# Debug a container with no shell

Pod `vault` in namespace `dojo-debug-ephemeral` is running normally. Its
container writes a token to `/run/vault/token` at startup, and that file
exists only inside the running container.

The image is distroless: no shell, no `cat`, no package manager. `kubectl exec`
has nothing to exec.

Recover the token **without restarting or recreating the Pod**, then store it
in the same namespace as a Secret:

- Secret name `recovered`
- key `token`, value the token you found

Along the way, attach an ephemeral container named `sleuth` to the Pod, using
image `busybox:1.36`, targeting the `vault` container. It must still be
attached when you finish.

Read the flags before you run anything: an ephemeral container cannot be
removed, renamed or reconfigured once attached, so a wrong first attempt is
permanent for the life of the Pod.

Answer each checkpoint with `dojo check <answer>`, and `dojo check` on its own
for the configuration step.
