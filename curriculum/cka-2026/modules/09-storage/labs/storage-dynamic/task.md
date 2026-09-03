# Let the cluster create the volume for you

Every other storage exercise here hands you a PersistentVolume that someone
made by hand. This cluster can do better: it runs a provisioner that creates
volumes on demand. Find it with `kubectl get storageclass`.

StorageClass `dojo-dynamic` already exists, but as configured it cannot
provision anything — it names no real provisioner, and it binds immediately
rather than waiting to see where the workload lands.

In namespace `dojo-storage-dynamic`:

1. Make `dojo-dynamic` actually provision, using the same provisioner the
   cluster's existing class uses, and make it defer binding until something
   wants to use the volume. Keep the `Delete` reclaim policy.
2. Create PVC `scratch` requesting `1Gi`, `ReadWriteOnce`, class
   `dojo-dynamic`. Look at its phase before going further, and note the reason
   `kubectl describe` gives.
3. Create Pod `writer` — image `busybox:1.36`, command `sleep 3600` — mounting
   that claim at `/data`.

By the end the claim must be `Bound` to a PersistentVolume that you never
created.
