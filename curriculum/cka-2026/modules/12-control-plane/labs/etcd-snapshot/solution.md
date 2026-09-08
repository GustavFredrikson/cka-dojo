# Worked solution

## Working it out

Everything you need is in the manifest kubelet is already using:

```bash
ssh cp1
sudo grep -E 'listen-client-urls|cert-file|key-file|trusted-ca-file|data-dir' \
  /etc/kubernetes/manifests/etcd.yaml
```

On a kubeadm cluster that gives you:

- endpoint `https://127.0.0.1:2379`
- CA `/etc/kubernetes/pki/etcd/ca.crt`
- client certificate `/etc/kubernetes/pki/etcd/server.crt`
- key `/etc/kubernetes/pki/etcd/server.key`
- data directory `/var/lib/etcd`

Reading them off the manifest rather than reciting them is the habit worth
building: on a cluster that was not built by kubeadm, the recited paths are
wrong and the manifest is still right.

`healthcheck-client.crt` works just as well as `server.crt` here. Either is a
client certificate signed by the etcd CA, which is all etcd asks for.

## Fixing it

```bash
sudo etcdctl \
  --endpoints=https://127.0.0.1:2379 \
  --cacert=/etc/kubernetes/pki/etcd/ca.crt \
  --cert=/etc/kubernetes/pki/etcd/server.crt \
  --key=/etc/kubernetes/pki/etcd/server.key \
  snapshot save /opt/dojo/etcd-snapshot.db
```

`sudo mkdir -p /opt/dojo` first if it is not there.

`ETCDCTL_API=3` is the default from etcd 3.4 onwards, so setting it is
harmless but no longer required. Set it anyway if you are unsure which client
version is installed -- on the exam you cannot assume.

## Checking before you grade

```bash
sudo etcdutl snapshot status /opt/dojo/etcd-snapshot.db -w table
```

Note which binary this is. From etcd 3.6, the offline snapshot subcommands
(`status`, `restore`) live in **etcdutl**, not etcdctl; on 3.5 and earlier
both work and etcdctl prints a deprecation warning. `snapshot save` is the
opposite case -- it talks to a running server, so it stays in `etcdctl`.

This cluster runs etcd 3.6, and the way etcdctl declines is worth seeing once,
because it does not look like a refusal:

```bash
sudo etcdctl snapshot status /opt/dojo/etcd-snapshot.db
```

It prints the usage text for `etcdctl snapshot` and **exits 0**. No error, no
non-zero status -- the subcommand simply is not there any more, and an
unrecognised one falls through to help. In a script that means
`etcdctl snapshot status ... || fallback` never takes the fallback, and you
end up parsing help output. Reach for `etcdutl` when the operation is offline,
and check that what came back is what you expected rather than trusting an
exit code.

A snapshot of a real cluster reports several hundred keys and a few megabytes.
A total key count in single figures means you snapshotted something else, or
an etcd that is not the cluster's.
