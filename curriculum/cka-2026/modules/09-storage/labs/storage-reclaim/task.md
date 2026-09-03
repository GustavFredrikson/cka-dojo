# Keep the volume when the claim goes away

PersistentVolume `dojo-storage-reclaim` (`2Gi`, `ReadWriteOnce`, class
`dojo-static-reclaim`) is going to hold an audit archive. As it is configured
right now, deleting the claim that uses it would also destroy the volume and
everything in it.

That is the wrong behaviour for this data.

1. Reconfigure the volume so that losing its claim does **not** destroy it.
2. In namespace `dojo-storage-reclaim`, create PVC `archive` requesting `1Gi`
   with the matching access mode and class, and confirm it binds.
3. Delete the claim.

When you are finished the volume must still exist, and must be holding on to
its data rather than having been cleaned up.
