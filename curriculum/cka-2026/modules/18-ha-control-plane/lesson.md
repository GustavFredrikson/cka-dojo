# Highly-available control plane

## Mental model

Three control planes, each running its own etcd member as a static Pod
("stacked" etcd, as opposed to an external etcd cluster), behind one stable
endpoint that clients and joining nodes both use.

The endpoint is the load-bearing idea. If a client pointed at cp1's address,
losing cp1 would cost you the cluster no matter how many control planes there
were. `--control-plane-endpoint` is written into the API server certificate's
SANs, into kube-proxy's ConfigMap and into every kubeconfig at `kubeadm init`
time — which is why it cannot be added or changed afterwards without reissuing
certificates, and why it is a name here rather than an address.

The three API servers are stateless and interchangeable; the load balancer
round-robins between them. The scheduler and controller-manager all run too, but
lease-elect a single leader.

## Objects involved

- `--control-plane-endpoint`, and the `kubeadm-config` ConfigMap that records it.
- The load balancer (haproxy here) and the `/etc/hosts` entry that resolves it.
- etcd members, and the certificate key from `kubeadm init phase upload-certs`.
- Leader-election Leases in `kube-system`.

## Commands worth knowing

```bash
kubectl config view --minify
kubectl -n kube-system get configmap kubeadm-config -o yaml
kubectl -n kube-system get pods -l component=kube-apiserver

sudo etcdctl --endpoints=https://127.0.0.1:2379 \
  --cacert=/etc/kubernetes/pki/etcd/ca.crt \
  --cert=/etc/kubernetes/pki/etcd/server.crt \
  --key=/etc/kubernetes/pki/etcd/server.key \
  member list -w table
sudo etcdctl ... endpoint health --cluster
sudo etcdctl ... endpoint status --cluster -w table

sudo kubeadm init phase upload-certs --upload-certs   # mints a certificate key
sudo kubeadm token create --print-join-command
```

## Diagnostic workflow

Establish how far a failure reaches before changing anything. Run the same
command from a control plane's own admin kubeconfig: if that works and the
workstation does not, the cluster is healthy and the endpoint is not.

Do not trust `kubectl get nodes` for etcd health. A stopped member leaves the
remaining two forming a majority of three, so everything looks normal while the
cluster is one failure from losing quorum. Ask etcd directly.

## Quorum arithmetic

etcd needs a strict majority to accept a write:

| members | majority | can lose |
|---|---|---|
| 3 | 2 | 1 |
| 2 | 2 | 0 |
| 5 | 3 | 2 |

The consequence that catches people: **removing** a member is more dangerous
than stopping one. A stopped member still counts toward the membership, so 3
members with one down is still a majority of 3. Remove it and you have a
two-member cluster that tolerates nothing. Even counts are always worse than the
odd number below them, which is why etcd clusters are 3, 5 or 7.

## Common CKA failure modes

- Repointing a kubeconfig at one control plane to "fix" a broken endpoint, which
  works and quietly discards the high availability.
- `etcdctl member remove` on a member that was merely stopped, turning a
  five-second fix into a rebuild.
- Reading node Ready status as etcd health.
- Trying to add `--control-plane-endpoint` to a cluster that was built without
  one, without reissuing certificates.
- Joining a control plane with an expired certificate key — it lasts two hours,
  against the bootstrap token's twenty-four.
- Forgetting that the load balancer is itself a single point of failure.

## 5-minute walkthrough

```bash
kubectl config view --minify
grep k8s-api /etc/hosts
kubectl get nodes
ssh cp1 'sudo etcdctl ... endpoint health --cluster'
```

## Labs

```text
dojo start ha-topology
dojo start ha-lost-member
dojo start ha-endpoint-down
```
