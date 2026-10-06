# Worked solution

## The shape of it

```
kubectl  ->  k8s-api:6443  ->  haproxy on terminal  ->  cp1 | cp2 | cp3
                                                        each with a local
                                                        etcd member
```

**The endpoint is a name.** `kubectl config view --minify` shows
`server: https://k8s-api:6443`, and `grep k8s-api /etc/hosts` resolves it to the
workstation. That indirection is the whole design: if the client pointed at
cp1's address, losing cp1 would cost you the cluster no matter how many control
planes existed.

**haproxy** is what listens there. It is a plain TCP proxy — it does not
terminate TLS, it forwards the connection, so the API server's certificate is
what the client verifies. `ss -ltn` on the workstation shows it bound to 6443.

**The cluster knows the endpoint too.** `kubeadm-config` in `kube-system` records
`controlPlaneEndpoint: k8s-api:6443`, which is how `kubeadm join` learns where to
point a new node without being told. It is also why the endpoint cannot be
changed casually: it is baked into the API server certificate's SANs at
`kubeadm init`, and changing it means reissuing certificates.

## Stacked etcd, and the quorum arithmetic

Each control plane runs its own etcd member as a static Pod — "stacked", as
opposed to an external etcd cluster on separate machines.

```bash
sudo etcdctl --endpoints=https://127.0.0.1:2379 \
  --cacert=/etc/kubernetes/pki/etcd/ca.crt \
  --cert=/etc/kubernetes/pki/etcd/server.crt \
  --key=/etc/kubernetes/pki/etcd/server.key \
  member list -w table
```

Three members. etcd needs a strict majority to accept a write, so:

| members | majority | can lose |
|---|---|---|
| 3 | 2 | **1** |
| 2 | 2 | 0 |
| 5 | 3 | 2 |

This is the number worth carrying into the exam. Three tolerates one loss. Two
tolerates none — which is why *removing* a member from a three-member cluster is
more dangerous than merely stopping one: a stopped member still counts toward
the membership, and the remaining two still form a majority of three. Remove it
and you have a two-member cluster that cannot lose anything at all.

Even counts are worse than the odd number below them, which is why etcd clusters
are always 3, 5 or 7.

## Three API servers, one endpoint

All three run simultaneously and are stateless — they all talk to the same etcd
cluster, so it does not matter which one answers. haproxy round-robins between
them. The scheduler and controller-manager are different: all three run, but
they lease-elect a leader and only the leader acts.

```bash
kubectl -n kube-system get lease -l apiserver.kubernetes.io/identity
kubectl -n kube-system get lease kube-scheduler -o yaml | grep holderIdentity
```
