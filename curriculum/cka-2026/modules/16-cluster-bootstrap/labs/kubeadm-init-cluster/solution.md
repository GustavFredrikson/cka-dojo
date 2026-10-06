# Worked solution

## Working it out

`kubeadm init` needs three things it cannot safely guess, and this environment
makes all three matter.

**Which address to advertise.** Every certificate the API server gets, every
kubeconfig kubeadm writes, and the `server:` URL the workstation will use are
all built from this. Get it wrong and the cluster works from cp1 and from
nowhere else. Finding that address has a trap in it that is worth meeting once. The obvious
command is wrong:

```bash
$ grep cp1 /etc/hosts
127.0.1.1 cp1
192.168.104.7 cp1
```

Ubuntu writes `127.0.1.1 <hostname>` into `/etc/hosts` by default, and it comes
first. Pass that to kubeadm and you get:

```
error: unable to select an IP from lo network interface
```

Take it from somewhere unambiguous instead — the kubelet already has it pinned:

```bash
sed -n 's/.*--node-ip=\([0-9.]*\).*/\1/p' /etc/default/kubelet
ip -4 -o addr show | grep 192.168.104     # or straight off the interface
```

Not the default route either: on this provider every VM has the same address
there.

**Which pod CIDR.** The controller-manager only allocates per-node pod CIDRs if
you pass `--pod-network-cidr`, and the CNI you install later has to agree with
whatever you chose. `10.244.0.0/24` appearing on the Node's `.spec.podCIDR` is
how you confirm it took.

**Which CRI socket.** `/etc/crictl.yaml` names it.

## Fixing it

```bash
ssh cp1
sudo kubeadm init \
  --apiserver-advertise-address=$(grep -w cp1 /etc/hosts | awk '{print $1}') \
  --pod-network-cidr=10.244.0.0/16 \
  --service-cidr=10.96.0.0/12 \
  --cri-socket=unix:///run/containerd/containerd.sock
```

It takes two to four minutes. Watch what it prints: the phases it names
(certs, kubeconfig, control-plane, etcd, wait-control-plane) are exactly the
directories it fills in, and the join command at the end is what the next
exercise needs.

Then get a kubeconfig onto the workstation:

```bash
exit
mkdir -p ~/.kube
ssh cp1 'sudo cat /etc/kubernetes/admin.conf' > ~/.kube/config
chmod 600 ~/.kube/config
kubectl get nodes
```

`admin.conf` already points at the advertise address, so it works unchanged from
the workstation. That is only true because you advertised a routable address —
had you used `127.0.0.1`, you would now be editing the `server:` field.

## What you should see, and why it looks wrong

```
NAME   STATUS     ROLES           AGE   VERSION
cp1    NotReady   control-plane   1m    v1.35.8
```

`NotReady` is correct at this point. The kubelet reports
`NetworkPluginNotReady` because nothing has written a CNI config to
`/etc/cni/net.d`. CoreDNS will sit `Pending` for the same reason — it needs a
pod network, and there is not one. Everything else in `kube-system` should be
`Running`.

## If it goes wrong

```bash
sudo kubeadm reset -f --cri-socket=unix:///run/containerd/containerd.sock
sudo rm -rf /etc/cni/net.d /root/.kube
```

puts the node back so you can try again. `kubeadm reset` is not destructive to
anything but the cluster, and on this profile that is the whole point.

## Checking before you grade

```bash
kubectl get nodes -o wide
kubectl -n kube-system get pods
kubectl get node cp1 -o jsonpath='{.spec.podCIDR}'; echo
ls /etc/kubernetes/manifests
```

`.spec.podCIDR` must be `10.244.0.0/24` — the first /24 out of the /16 you
passed. If it is empty, `--pod-network-cidr` was not passed and the CNI will not
work later.
