# Worked solution

## What the plan tells you, and what it does not

```bash
ssh cp1
sudo kubeadm upgrade plan
```

The output is shorter than you might expect, and the reason is the lesson:

```
[upgrade/versions] Cluster version: 1.34.11
[upgrade/versions] kubeadm version: v1.34.11
[upgrade/versions] Target version: v1.34.11
[upgrade/versions] Latest version in the v1.34 series: v1.34.11
```

It offers you **1.34.11 — the version you are already on** — and says nothing
about 1.35 at all.

That is not a bug and it is the single most useful thing in this exercise.
`kubeadm upgrade plan` will not plan beyond its own minor: the binary is
1.34.11, so it looks at the 1.34 series and stops. The line
`remote version is much newer: v1.37.0; falling back to: stable-1.34` is it
telling you exactly that.

So the order is forced, and it is the order the real procedure uses:

1. re-point the apt repository to the new minor,
2. upgrade the **`kubeadm` package** first, on its own,
3. *then* `kubeadm upgrade plan` shows you 1.35, and `kubeadm upgrade apply`
   will run it.

Once you are on a 1.35 kubeadm, the plan output grows a per-component table and
a note that is easy to skim past:

> _Before you can perform this upgrade, you have to update the kubelet on each
> node._

That is the part `kubeadm upgrade` will not do for you, on any node, ever. It
manages the control-plane components because they are static Pods it wrote; the
kubelet is an OS package and stays yours.

## The two obstacles

```bash
apt-mark showhold
# kubeadm
# kubectl
# kubelet

cat /etc/apt/sources.list.d/kubernetes.list
# deb [...] https://pkgs.k8s.io/core:/stable:/v1.34/deb/ /
```

The holds exist so that an unrelated `apt-get upgrade` cannot move your cluster
a minor version without asking — which is a real failure mode, and why you
should put them back afterwards.

The repository is pinned to one minor because that is how the Kubernetes apt
repositories are published: `core:/stable:/v1.34`, `core:/stable:/v1.35`, and so
on. There is no "latest" repository. Upgrading across a minor therefore always
means editing this file, and `apt-cache madison kubeadm` will keep showing you
only 1.34 versions until you do.

## Version skew, and why order matters

A kubelet may run up to **three** minor versions behind the API server. It may
never run *ahead* of it. Everything else follows from that:

- the control plane goes first, always;
- workers follow, one at a time, drained and uncordoned around the kubelet step;
- leaving a worker a minor behind for a while is fine and supported.

`kubectl` itself is supported one minor either side, which is why upgrading it
alongside the kubelet is conventional rather than required.

## Nothing here changed anything

That is deliberate. `upgrade-control-plane` and `upgrade-worker` consume this
environment — a kubeadm upgrade is one-way and nothing can put 1.35 back to
1.34, so those two run once per `dojo env reset`. This exercise reads only, so
you can come back to it as often as you like.
