# Worked solution

## The shape of it

Three layers, and the exam expects you to know which one you are looking at:

```
kubectl  ->  API server  ->  kubelet  ->  CRI  ->  containerd  ->  runc
```

`kubectl` needs the API server. `crictl` speaks CRI to the kubelet's own
runtime socket and needs nothing above it. `ctr` is containerd's native client
and does not speak CRI at all — it can see containerd's namespaces directly,
including images the CRI layer does not expose the same way.

That is the whole reason this lab exists: when the API server is down,
`crictl ps` and `crictl logs` are how you find out why.

## The answers

**`/etc/crictl.yaml`**, pointing at **`/run/containerd/containerd.sock`**.
Without that file `crictl` guesses, and on a node with more than one runtime
socket it guesses wrong. This environment writes it during provisioning.

**`containerd://2.3.5`** is what the Node object reports, from
`.status.nodeInfo.containerRuntimeVersion`:

```bash
kubectl get node cp1 -o jsonpath='{.status.nodeInfo.containerRuntimeVersion}'
```

The kubelet queried the runtime over CRI and put the answer in the Node status,
which is how the API server knows something it never talks to directly.

**The sandbox.** `crictl pods` lists sandboxes, `crictl ps` lists containers,
and the counts differ because every Pod has exactly one sandbox plus one or
more containers. The sandbox is created first and holds the network namespace —
the IP belongs to it, not to any container in the Pod, which is why containers
in a Pod share an address and can reach each other on localhost.

**`registry.k8s.io/pause:3.10.1`** is the image the sandbox runs. It does
nothing but sleep and reap orphaned children. In `/etc/containerd/config.toml`:

```bash
grep -i sandbox /etc/containerd/config.toml
```

The version must match what kubeadm expects, which is why provisioning here
reads it from `kubeadm config images list` rather than hardcoding it.

**`systemd`**, in both places — `cgroupDriver: systemd` in
`/var/lib/kubelet/config.yaml` and `SystemdCgroup = true` in containerd's
config. If they disagree, Pods start and then die, and the symptom points at
everything except the cause. On a node you did not build, this is worth
checking early.

**`kubectl`** is the one that stops working. Both others talk to the node.

## Worth knowing beyond the checkpoints

`crictl` sees things `kubectl` cannot: exited containers from a Pod that has
been recreated, sandboxes whose containers all failed, and the control-plane
static Pods while the API server is down.

```bash
sudo crictl ps -a --name kube-apiserver     # including exited
sudo crictl logs <container-id>
sudo crictl inspect <container-id>
```

`crictl logs` takes a *container* id from `crictl ps`, not a Pod id from
`crictl pods` — a small thing that costs time under pressure.
