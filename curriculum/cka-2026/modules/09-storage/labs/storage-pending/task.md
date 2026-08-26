# A claim never binds

PVC `data` in namespace `dojo-storage-pending` remains Pending. It should bind
to the designated static PV `dojo-storage-pending`.

Use claim/PV state and Events to identify the incompatible constraint. Restore
binding while preserving the object names and at least a `512Mi` request.
