# Worked solution

Compare these fields on PV and PVC: `storageClassName`, capacity/request,
`accessModes`, `volumeMode`, selectors and current claim reference. The
variants mismatch class, capacity or access mode. Recreate the Pending claim
with compatible values, then confirm `Bound` and the selected `volumeName`.
