# Repair a known bad container command

Deployment `web` in namespace `dojo-workloads-guided` is unavailable because
its Pod template overrides the image's normal command with one that exits.

Use Pod status and previous logs to observe the failure, remove the bad command
override, and restore both replicas. Preserve the Deployment and image.
