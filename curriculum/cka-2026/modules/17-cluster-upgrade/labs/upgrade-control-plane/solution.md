# Worked solution

## Working it out

A node's Kubernetes version lives in three places, and an upgrade has to move
all three:

1. the **`kubeadm` binary** — an apt package;
2. the **control-plane component images**, which live in the static Pod
   manifests under `/etc/kubernetes/manifests` and are rewritten by
   `kubeadm upgrade apply`;
3. the **`kubelet`**, another apt package, which `kubeadm upgrade` does *not*
   touch.

Moving one and not the others is the usual way this goes wrong, and it is why
grading checks all three separately.

Before any of that, two things block you, both deliberately:

```bash
apt-mark showhold
cat /etc/apt/sources.list.d/kubernetes.list
```

The packages are held, and the repository carries exactly one minor —
`v1.34`. The Kubernetes apt repositories are per-minor by design, so upgrading
across a minor always means re-pointing the list. This is exactly the shape of
the task on the exam.

## Fixing it

Repository and kubeadm first:

```bash
ssh cp1
sudo sed -i 's#/v1\.34/#/v1.35/#' /etc/apt/sources.list.d/kubernetes.list
sudo apt-get update
apt-cache madison kubeadm | head -3
```

Pick the version string it shows (something like `1.35.8-1.1`), then:

```bash
sudo apt-mark unhold kubeadm
sudo apt-get install -y kubeadm=1.35.8-1.1
sudo apt-mark hold kubeadm
kubeadm version -o short
```

Ask before acting — `upgrade plan` tells you what it will do and what it will
leave to you:

```bash
sudo kubeadm upgrade plan
sudo kubeadm upgrade apply v1.35.8 -y
```

That rewrites the static Pod manifests, and the kubelet restarts each component
as its file changes. It also prints, explicitly, that you must upgrade the
kubelet yourself.

Now the kubelet, with the node drained. Run the drain from the workstation,
where you have a kubeconfig:

```bash
kubectl drain cp1 --ignore-daemonsets
```

then on cp1:

```bash
sudo apt-mark unhold kubelet kubectl
sudo apt-get install -y kubelet=1.35.8-1.1 kubectl=1.35.8-1.1
sudo apt-mark hold kubelet kubectl
sudo systemctl daemon-reload
sudo systemctl restart kubelet
```

and back on the workstation:

```bash
kubectl uncordon cp1
kubectl get nodes
```

## The two things people forget

**The uncordon.** `kubectl drain` cordons the node and nothing un-cordons it for
you. A node left `SchedulingDisabled` is a silently degraded cluster, and it is
the single most common omission in this task. Grading checks it.

**Re-holding the packages.** You lifted the holds to install; put them back, or
the next unrelated `apt-get upgrade` moves your cluster a minor without asking.

## On version skew

Leaving `worker1` on 1.34 while cp1 runs 1.35 is correct and supported: a
kubelet may be up to three minors behind the API server, never ahead. That is
why the control plane goes first — upgrading a worker past its API server is the
unsupported direction.

## Checking before you grade

```bash
kubectl get nodes -o wide
ssh cp1 'kubeadm version -o short; apt-mark showhold'
ssh cp1 'grep image: /etc/kubernetes/manifests/kube-apiserver.yaml'
kubectl -n dojo-upgrade-control-plane get deploy ledger
```

`kubectl get nodes` should show cp1 on 1.35 and worker1 on 1.34, both Ready,
neither `SchedulingDisabled`.

## Why this lab cannot be reset

`kubeadm upgrade` migrates cluster configuration and etcd data forward. There is
no downgrade path, and no fault `undo` can synthesise one — so the drill is
genuinely one-shot per environment build. The lab opens with a version guard
that refuses to run on an already-upgraded cluster rather than handing you a
lab that grades as passed before you start. `dojo env reset --profile
upgrade-1.34` rebuilds it, in four to eight minutes.
