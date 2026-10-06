# Worked solution

## What you are looking at

A machine with every Kubernetes package installed and nothing configured. The
distinction matters because the two halves fail differently: a missing package
is an install problem, and a missing config is a `kubeadm` problem.

**`systemctl is-active kubelet`** reports `activating`, because the unit is
enabled and the kubelet cannot start. It exits, systemd restarts it, and it
exits again. That loop is normal here and alarming if you do not expect it.

**`journalctl -u kubelet`** says why: it cannot find
`/etc/kubernetes/kubelet.conf` (and on some paths `/var/lib/kubelet/config.yaml`).
Both are written by `kubeadm init`. Until then the kubelet has no API server to
register with and no configuration to run from.

This is worth internalising for the exam: a crash-looping kubelet on a node that
has never joined a cluster is *expected*, and chasing it as a fault wastes time.
The same symptom on a node that *was* working is a real problem.

**`swapon --show`** prints nothing — swap is off, so kubeadm's swap preflight
check will pass. On a machine you did not prepare, this is one of the first
things `kubeadm init` complains about.

**Port 6443** is what the API server binds, and `kubeadm init` preflight-checks
that it is free. If something else holds it — usually a previous, half-removed
cluster — init refuses.

**`--node-ip`** in `/etc/default/kubelet` pins this node's address. Without it
the kubelet picks whatever interface it likes, and on this provider every VM has
the same address on the default route — so all three nodes would register as the
same address and the cluster would be quietly wrong.

**`/run/containerd/containerd.sock`** is the CRI socket, from `/etc/crictl.yaml`.
You pass it to `kubeadm init --cri-socket`. On a node with exactly one runtime
kubeadm can detect it, but stating it is free and removes a class of ambiguity.

## What comes next

`kubeadm init` writes, in order: the certificate authority and every
certificate under `/etc/kubernetes/pki`, four kubeconfigs in
`/etc/kubernetes`, and four static Pod manifests in
`/etc/kubernetes/manifests`. The kubelet is watching that last directory, which
is how the control plane starts without a control plane to schedule it.
