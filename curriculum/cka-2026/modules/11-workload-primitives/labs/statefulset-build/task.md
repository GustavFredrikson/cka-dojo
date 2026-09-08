# A StatefulSet with per-Pod storage

Work in namespace `dojo-statefulset-build`. StorageClass `dojo-statefulset`
already exists.

Create:

**1. A headless Service named `db`** on port `5432`, selecting `app: db`.

**2. A StatefulSet named `db`** with two replicas, governed by that Service:

- image `busybox:1.36`, command `sleep 3600`
- label `app: db`
- a volume claim template named `data`, requesting `1Gi`, `ReadWriteOnce`,
  storage class `dojo-statefulset`, mounted at `/data`

Both Pods must be Ready, both claims Bound, and `db-0` must be able to resolve
`db-1` by name through the Service.
