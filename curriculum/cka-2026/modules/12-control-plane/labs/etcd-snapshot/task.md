# Back up etcd

This cluster runs stacked etcd: a static Pod on `cp1`, with its data directory
and its TLS material on that node's filesystem.

On `cp1`, save a snapshot of the live etcd to:

```
/opt/dojo/etcd-snapshot.db
```

Use the v3 API and authenticate with the certificates etcd itself uses. Do not
guess the endpoint or the certificate paths -- read them out of the static Pod
manifest.

Then confirm the snapshot is valid and contains the cluster's keys, without
restoring it.
