# A machine that is not yet a node

`cp1` has containerd running, and the kubelet, kubeadm and kubectl installed at
the version this environment pins. Swap is off and the sysctls are set. What it
does not have is a cluster.

Nothing here is broken and nothing you do should change anything. Read the
machine as it is, and answer each checkpoint with `dojo check <answer>`.

```bash
ssh cp1
systemctl is-active kubelet
journalctl -u kubelet -n 20 --no-pager
swapon --show
cat /etc/default/kubelet
cat /etc/crictl.yaml
kubeadm version -o short
```

This is the state every `kubeadm init` starts from, and recognising it is worth
more than memorising the command.
