# What the kubelet actually talks to

Nothing is broken here, and nothing you do should change that.

The kubelet does not run containers. It asks a runtime to, over a socket,
through an interface — and when the API server is down, that socket is the only
window you have into what the control plane is doing.

Work on `cp1` and answer each checkpoint with `dojo check <answer>`.

Useful starting points:

```bash
cat /etc/crictl.yaml
sudo crictl pods
sudo crictl ps
sudo crictl info
grep -i sandbox /etc/containerd/config.toml
grep -i cgroupDriver /var/lib/kubelet/config.yaml
```

`crictl` needs root on this node.
