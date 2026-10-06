# Building a cluster

## Mental model

Every other module starts with a cluster; this one starts with three Linux
machines. They are prepared but not initialised: containerd runs, the kubelet,
kubeadm and kubectl are installed at the pinned version, swap is off and the
sysctls are set. What does not exist is Kubernetes.

`kubeadm init` writes certificates, kubeconfigs and four static Pod manifests,
and then the kubelet — which has been crash-looping for want of a config —
reads that directory and brings up the control plane it is about to register
with. That is how a cluster starts without a cluster: nothing schedules the
control plane, the kubelet runs it off local disk.

A fresh cluster is `NotReady` on purpose. Kubernetes defines a network interface
and ships no implementation; until something writes a config into
`/etc/cni/net.d`, the kubelet reports the network as not ready.

## Objects involved

- The cluster CA and the certificates under `/etc/kubernetes/pki`.
- The four kubeconfigs in `/etc/kubernetes`, each embedding a client certificate.
- Static Pod manifests in `/etc/kubernetes/manifests`, and their mirror Pods.
- Bootstrap tokens and the discovery CA cert hash.
- The CNI DaemonSet, and the config it writes to `/etc/cni/net.d`.
- Node objects, and the `.spec.podCIDR` the controller-manager allocates.

## Commands worth knowing

```bash
grep cp1 /etc/hosts
sudo kubeadm init \
  --apiserver-advertise-address=<cp1 address> \
  --pod-network-cidr=10.244.0.0/16 \
  --service-cidr=10.96.0.0/12 \
  --cri-socket=unix:///run/containerd/containerd.sock
mkdir -p ~/.kube && ssh cp1 'sudo cat /etc/kubernetes/admin.conf' > ~/.kube/config
kubectl apply -f https://github.com/flannel-io/flannel/releases/latest/download/kube-flannel.yml
sudo kubeadm token create --print-join-command
sudo kubeadm join <addr>:6443 --token ... --discovery-token-ca-cert-hash sha256:...
sudo kubeadm reset -f --cri-socket=unix:///run/containerd/containerd.sock
```

## Diagnostic workflow

Work down the stack. Is the kubelet running, and if not, what does
`journalctl -u kubelet` say it cannot find? Is the API server answering
(`kubectl get --raw=/readyz`)? Is the node Ready, and if not, does its Ready
condition name the network? Does a Pod's address fall inside the node's
`.spec.podCIDR`?

When `kubeadm init` fails, read which *phase* it failed in — preflight, certs,
kubeconfig, control-plane, wait-control-plane. That is usually the whole
diagnosis.

## Common CKA failure modes

- Swap left on, so preflight refuses.
- `--pod-network-cidr` omitted: no node CIDRs are ever allocated.
- A CNI whose range disagrees with `--pod-network-cidr`: Ready node,
  unroutable Pods, and it surfaces much later as "some Pods cannot reach
  each other".
- `--apiserver-advertise-address` on the wrong interface: works on the control
  plane, unreachable everywhere else, and baked into the certificates.
- Joining with an expired token — they last 24 hours.
- Joining a node that still has `/etc/kubernetes/kubelet.conf` from before.
- Copying `admin.conf` to a workstation when its `server:` is `127.0.0.1`.
- Treating a NotReady fresh cluster, or a crash-looping kubelet on a machine
  that has never joined, as a fault rather than the expected state.

## 5-minute walkthrough

```bash
ssh cp1
systemctl is-active kubelet
journalctl -u kubelet -n 20 --no-pager
cat /etc/default/kubelet /etc/crictl.yaml
ls /etc/kubernetes/manifests
```

## Labs

```text
dojo start kubeadm-preflight
dojo start kubeadm-init-cluster
dojo start cni-install
dojo start kubeadm-join-worker
```
