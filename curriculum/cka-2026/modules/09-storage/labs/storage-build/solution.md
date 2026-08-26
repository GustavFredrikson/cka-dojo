# Worked solution

Write a PVC matching the PV's class, capacity and access mode, then a Pod whose
`persistentVolumeClaim.claimName` is `cache`. Verify `Bound` and inspect
`spec.volumeName` before checking the Pod.
