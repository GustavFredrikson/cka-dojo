# Worked solution

## Working it out

```bash
kubectl -n dojo-kubeadm-join-worker describe pod -l app=batch | tail -10
```

`0/1 nodes are available: 1 node(s) didn't match Pod's node affinity/selector`.
The Deployment has `nodeSelector: kubernetes.io/hostname: worker1`, and:

```bash
kubectl get nodes
```

shows only `cp1`. The node the workload wants does not exist as far as the
cluster is concerned — the machine is running, it has simply never joined.

## Fixing it

A worker joins with `kubeadm join`, and that command needs two credentials that
can only come from the control plane: a bootstrap token, and a hash of the
cluster CA so the joining node can verify what it is talking to.

Mint a fresh one rather than hunting for the one `kubeadm init` printed —
bootstrap tokens expire after 24 hours, and on the exam you will not have the
original output:

```bash
ssh cp1
sudo kubeadm token create --print-join-command
```

It prints the whole thing:

```
kubeadm join 192.168.104.7:6443 --token abcdef.0123456789abcdef \
  --discovery-token-ca-cert-hash sha256:1234...
```

Run it on worker1, as root, adding the CRI socket:

```bash
ssh worker1
sudo kubeadm join 192.168.104.7:6443 --token abcdef.0123456789abcdef \
  --discovery-token-ca-cert-hash sha256:1234... \
  --cri-socket=unix:///run/containerd/containerd.sock
```

Then, from the workstation:

```bash
kubectl get nodes -w
kubectl -n dojo-kubeadm-join-worker get pods -o wide
```

`worker1` appears within seconds and goes Ready once the CNI DaemonSet has
placed a Pod on it. The `batch` Pods schedule on their own after that — nothing
needs restarting.

## What can go wrong

- **`kubelet.conf already exists`.** The node has joined something before.
  `sudo kubeadm reset -f` and try again.
- **The token has expired.** They last 24 hours. Mint another; this is the most
  common version of "the join command does not work".
- **`connection refused` to :6443.** You used the wrong address for the control
  plane. It is in `/etc/hosts`.
- **The node joins and stays NotReady.** It is waiting for the CNI DaemonSet to
  schedule a Pod on it. Give it a minute, then look at
  `kubectl -n kube-system get pods -o wide`.

## Checking before you grade

```bash
kubectl get nodes
kubectl -n dojo-kubeadm-join-worker get deploy batch
ssh worker1 'systemctl is-active kubelet; ls /etc/kubernetes/kubelet.conf'
```

`kubectl get nodes` must show both, and `worker1` must not be
`SchedulingDisabled`.
