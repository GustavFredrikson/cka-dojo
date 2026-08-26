# Repair a known StorageClass mismatch

PVC `data` in namespace `dojo-storage-guided` is Pending because its
`storageClassName` does not match available PV `dojo-storage-guided`.

Recreate the claim with the correct class while preserving its name, `512Mi`
request and `ReadWriteOnce` mode.
