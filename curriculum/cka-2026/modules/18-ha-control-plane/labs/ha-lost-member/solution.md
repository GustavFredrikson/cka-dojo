# Worked solution

## Working it out

The cluster works, so the failure is something the cluster can absorb — which
on a three-member etcd cluster means exactly one thing has gone.

```bash
kubectl get nodes
```

`cp2` is `NotReady`. Its kubelet has stopped reporting.

That is worth pausing on: a control-plane node going NotReady does not
necessarily mean the control plane on it is gone. The kubelet is what reports
node status *and* what runs the static Pods, so when it stops, both the
reporting and the components go with it — but the other two API servers keep
serving and etcd keeps its majority, so nothing visible breaks.

```bash
ssh cp2
systemctl status kubelet
```

`inactive (dead)` and, importantly, `masked`. A masked unit cannot be started
until it is unmasked, which is what makes `systemctl start kubelet` fail with a
confusing message.

Check what it cost you, from cp1:

```bash
sudo etcdctl --endpoints=https://127.0.0.1:2379 \
  --cacert=/etc/kubernetes/pki/etcd/ca.crt \
  --cert=/etc/kubernetes/pki/etcd/server.crt \
  --key=/etc/kubernetes/pki/etcd/server.key \
  endpoint health --cluster
```

Two healthy, one unreachable. Two of three is still a majority, so writes still
work — you are one failure away from losing that, and nothing in
`kubectl get nodes` conveys the urgency.

## Fixing it

```bash
ssh cp2
sudo systemctl unmask kubelet
sudo systemctl enable --now kubelet
```

The kubelet starts, reads `/etc/kubernetes/manifests`, and brings the etcd
member, API server, scheduler and controller-manager back. The member rejoins
on its own — its data directory is intact and it catches up from the leader.

Confirm membership rather than trusting the node's Ready status:

```bash
sudo etcdctl ... endpoint health --cluster
kubectl -n kube-system get pods -l component=kube-apiserver
```

## What not to do

`etcdctl member remove` followed by `member add`. The member has not been
evicted — it is temporarily unreachable, and its data is fine. Removing it takes
you to a two-member cluster that tolerates no failures at all, and re-adding it
requires wiping its data directory and letting it resync. That is the repair for
a genuinely lost machine, not for a stopped service, and reaching for it here
turns a five-second fix into a rebuild.

## Checking before you grade

```bash
kubectl get nodes
ssh cp2 'systemctl is-active kubelet; systemctl is-enabled kubelet'
ssh cp1 'sudo etcdctl ... endpoint health --cluster'
kubectl -n kube-system get pods -l component=kube-apiserver
```

All four nodes Ready, kubelet `active` and `enabled` (not `masked`), three
healthy etcd endpoints, three API server Pods.
