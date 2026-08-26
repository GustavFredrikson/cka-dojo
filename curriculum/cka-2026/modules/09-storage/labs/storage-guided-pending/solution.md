# Worked solution

Export the PVC, remove generated metadata/status, set
`storageClassName: dojo-static-guided`, then delete and recreate it. Confirm
that `status.phase` becomes `Bound` and `spec.volumeName` names the intended PV.
