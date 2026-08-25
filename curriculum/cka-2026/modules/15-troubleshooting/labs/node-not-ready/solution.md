# Solution

## Working it out

Start where the symptom is visible:

```
kubectl get nodes
kubectl describe node worker1
```

`describe` shows the conditions, and for a node that has gone quiet you will
see `NodeStatusUnknown` -- "Kubelet stopped posting node status". That tells
you the answer is on the node, not in the API server.

The workstation is not part of the cluster, so get onto the node itself:

```
ssh worker1
```

Two services have to be healthy for a worker to work:

```
systemctl status kubelet
systemctl status containerd
```

- **kubelet is dead** -> nothing posts node status at all.
- **containerd is dead** -> the kubelet is running but cannot talk to the
  container runtime. `journalctl -u kubelet` fills up with connection errors
  against `/run/containerd/containerd.sock`, and the node goes NotReady for a
  different reason with the same headline symptom.

`journalctl -u kubelet --no-pager -n 50` distinguishes the two in seconds.

## Fixing it

```
sudo systemctl start kubelet          # or containerd
sudo systemctl enable kubelet         # so it survives a reboot
```

`systemctl enable --now <unit>` does both in one step.

The task says the fix must survive a reboot, so `enable` is not optional here:
`systemctl is-enabled kubelet` should say `enabled`.

If containerd was the dead one, the kubelet recovers on its own once the
runtime is back; you do not need to restart it.

## Checking before you grade

```
systemctl is-active kubelet containerd
systemctl is-enabled kubelet containerd
kubectl get nodes
```

Give the node twenty or thirty seconds to report in before grading.
