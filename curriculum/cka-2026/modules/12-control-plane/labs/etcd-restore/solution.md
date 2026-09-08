# Worked solution

## Working it out

Three facts decide the whole procedure.

**A restore is offline.** `etcdutl snapshot restore` does not talk to a
server; it reads a file and writes a fresh data directory. Nothing may be
using that directory while it happens, and the API server must not be serving
from the old data afterwards.

**A restore writes a new directory, and refuses to overwrite.** You cannot
restore into `/var/lib/etcd` while it has contents. So you restore somewhere
new and then point etcd at it -- or you move the old directory aside first.
Restoring into a new path is safer, because the old data survives your mistake.

**The static Pod manifest is the only thing that knows where the data is.**
`/etc/kubernetes/manifests/etcd.yaml` names the directory twice: once in
`--data-dir`, and once in the `hostPath` volume that mounts it into the
container. Change one and not the other and etcd comes up on an empty
database, which looks exactly like a successful restore until you look for
your data.

## Fixing it

```bash
ssh cp1
```

Stop the control plane by taking the static Pod manifests out of kubelet's
directory. etcd is the one that matters; moving the API server out too is
optional and makes the sequence easier to reason about.

```bash
sudo mkdir -p /etc/kubernetes/manifests-off
sudo mv /etc/kubernetes/manifests/etcd.yaml /etc/kubernetes/manifests-off/
sudo crictl ps | grep etcd        # wait until it is gone
```

Restore into a new directory:

```bash
sudo etcdutl snapshot restore /opt/dojo/pre-incident.db \
  --data-dir=/var/lib/etcd-restored
```

On etcd 3.5 and earlier, `sudo ETCDCTL_API=3 etcdctl snapshot restore ...`
does the same job and warns that it is deprecated. From 3.6 the subcommand
only exists in `etcdutl`.

Point the manifest at it. In `/etc/kubernetes/manifests-off/etcd.yaml`, both
of these have to change:

```yaml
    - --data-dir=/var/lib/etcd-restored
...
  - hostPath:
      path: /var/lib/etcd-restored
      type: DirectoryOrCreate
    name: etcd-data
```

Then put it back and let kubelet start it:

```bash
sudo mv /etc/kubernetes/manifests-off/etcd.yaml /etc/kubernetes/manifests/
```

The other route is to keep the manifest untouched and move the data instead:

```bash
sudo mv /var/lib/etcd /var/lib/etcd-old
sudo etcdutl snapshot restore /opt/dojo/pre-incident.db --data-dir=/var/lib/etcd
```

Grading accepts either -- it looks at the recovered data and a healthy
cluster, not at which path you chose. The second route needs no manifest edit,
which is fewer things to get wrong under time pressure; the first leaves the
original data in place, which is worth more when the snapshot turns out to be
the wrong one.

## Checking before you grade

The control plane takes thirty seconds or so to settle. Give it that before
you conclude anything:

```bash
sudo crictl ps | grep -E 'etcd|apiserver'
kubectl get --raw=/readyz
kubectl get nodes
kubectl -n dojo-etcd-restore get configmap ledger -o yaml
```

If `kubectl` reports a connection refused for more than a minute, read the
containers' own logs -- the API server cannot tell you why etcd did not start:

```bash
sudo crictl ps -a | grep etcd
sudo crictl logs <container-id>
```

An empty `dojo-etcd-restore` namespace, or no namespace at all, on an
otherwise healthy cluster is the signature of the mismatch described above:
etcd started on a directory that is not the one you restored into.
