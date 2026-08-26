# A rollout never becomes Ready

Deployment `web` in namespace `dojo-workloads-probe` has two running Pods, but
the rollout does not become available and Service `web` has no ready backends.

Find and correct the workload configuration. Do not remove the readiness probe.
