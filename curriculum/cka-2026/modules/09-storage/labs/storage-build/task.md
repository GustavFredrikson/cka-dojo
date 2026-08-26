# Create a claim and consuming Pod

PV `dojo-storage-build` offers `2Gi`, `ReadWriteOnce`, class `dojo-static-build`.

In namespace `dojo-storage-build`, create PVC `cache` requesting `1Gi` with
matching mode/class. Create Pod `cache-user` (`busybox:1.36`, `sleep 3600`)
mounting that claim at `/cache`.
