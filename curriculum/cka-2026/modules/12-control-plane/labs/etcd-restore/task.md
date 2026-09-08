# Restore etcd from a snapshot

Namespace `dojo-etcd-restore` held a ConfigMap named `ledger`. Both were
deleted, and the data in that ConfigMap is not recorded anywhere else.

A snapshot of etcd taken shortly before the deletion is on `cp1` at:

```
/opt/dojo/pre-incident.db
```

Restore the cluster from that snapshot, so that namespace
`dojo-etcd-restore` and ConfigMap `ledger` exist again with their original
contents.

The cluster must be fully functional when you are done: the API server
serving requests, and all three nodes `Ready`.

Do not recreate the ConfigMap by hand. You do not know what was in it.
