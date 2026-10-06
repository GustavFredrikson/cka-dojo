# Cluster upgrades

## Mental model

A node's Kubernetes version lives in three places that have to move together:
the `kubeadm` binary (an apt package), the control-plane component images
(written into the static Pod manifests by `kubeadm upgrade apply`), and the
`kubelet` (another apt package, which kubeadm never touches). Moving one and not
the others is the usual way an upgrade goes wrong.

Two things deliberately block you on every node, and both are per-node: the
Kubernetes packages are held with `apt-mark hold`, and the apt repository
carries exactly one minor. There is no "latest" repository — upgrading across a
minor always means editing `/etc/apt/sources.list.d/kubernetes.list`.

Order follows from the skew policy. A kubelet may run up to three minors behind
the API server and must never run ahead of it, so the control plane goes first
and workers follow one at a time.

## Objects involved

- `/etc/apt/sources.list.d/kubernetes.list`, and the apt holds.
- The static Pod manifests under `/etc/kubernetes/manifests`.
- `.status.nodeInfo.kubeletVersion` on each Node.
- Node cordon state, eviction, and PodDisruptionBudgets.

## Commands worth knowing

```bash
sudo kubeadm upgrade plan
apt-mark showhold
sudo sed -i 's#/v1\.34/#/v1.35/#' /etc/apt/sources.list.d/kubernetes.list
apt-cache madison kubeadm
sudo apt-mark unhold kubeadm && sudo apt-get install -y kubeadm=1.35.8-1.1
sudo kubeadm upgrade apply v1.35.8      # first control-plane node only
sudo kubeadm upgrade node               # every other node
kubectl drain worker1 --ignore-daemonsets --delete-emptydir-data
sudo systemctl daemon-reload && sudo systemctl restart kubelet
kubectl uncordon worker1
```

## Diagnostic workflow

Check the three places separately, because they fail separately:
`kubeadm version -o short` for the binary, `grep image:
/etc/kubernetes/manifests/kube-apiserver.yaml` for the components, and
`kubectl get nodes -o wide` for the kubelet. A node that reports the new version
in `kubectl get nodes` but still runs old component images means
`kubeadm upgrade apply` was never run.

If a drain hangs, read what it says about the disruption budget and look at
whether replacement Pods are becoming Ready somewhere else.

## Common CKA failure modes

- Installing the new packages and never running `kubeadm upgrade`.
- Running `kubeadm upgrade apply` on a worker — it is `upgrade node` there.
- Forgetting the kubelet, which kubeadm tells you about and people skim past.
- Draining and never uncordoning. The most common omission by a distance.
- Leaving the holds off, so a later `apt-get upgrade` moves the cluster.
- Fixing the repository on one node and assuming it applied to the others.
- Reaching for `--force` or `--disable-eviction` when a drain is slow, which
  bypasses the disruption budget and turns a routine upgrade into an outage.

## 5-minute walkthrough

```bash
ssh cp1
kubeadm version -o short
cat /etc/apt/sources.list.d/kubernetes.list
apt-mark showhold
sudo kubeadm upgrade plan
```

## Labs

```text
dojo start upgrade-plan
dojo start upgrade-control-plane
dojo start upgrade-worker
```

Note that the last two consume the environment: a kubeadm upgrade is one-way,
so they run once per `dojo env reset --profile upgrade-1.34`. `upgrade-plan`
changes nothing and repeats freely.
