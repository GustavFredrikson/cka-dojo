# An application rollout never becomes available

Deployment `web` in namespace `dojo-workloads-broken` should run two replicas
of `nginx:1.27-alpine`, but its latest rollout never becomes available.

Determine the failure from Pod status, Events and logs, then restore the
existing Deployment. Preserve its name, image and replica count.
