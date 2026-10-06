# Worked solution

## Working it out

The procedure is the control-plane one with a single command changed, plus a
drain that actually does something this time.

`kubeadm upgrade apply` is for the *first* control-plane node — it decides the
cluster's new version and migrates configuration. Every other node, worker or
additional control plane, uses:

```bash
sudo kubeadm upgrade node
```

which reads the decision already made and updates this node's local
configuration. On a worker it writes no static Pod manifests, because a worker
has none — so the kubelet version is the only evidence the upgrade happened,
and that is what grading checks.

Both obstacles are per-node. Fixing the repository and the holds on cp1 did
nothing for worker1.

## Fixing it

On worker1:

```bash
sudo sed -i 's#/v1\.34/#/v1.35/#' /etc/apt/sources.list.d/kubernetes.list
sudo apt-get update
apt-cache madison kubeadm | head -3
sudo apt-mark unhold kubeadm
sudo apt-get install -y kubeadm=1.35.8-1.1
sudo apt-mark hold kubeadm
sudo kubeadm upgrade node
```

Then drain, from the workstation:

```bash
kubectl drain worker1 --ignore-daemonsets --delete-emptydir-data
```

This one is not instant. Three `ledger` replicas are running with a
PodDisruptionBudget of `minAvailable: 2`, so eviction proceeds one Pod at a
time and waits for a replacement to become Ready on the other node before
taking the next. That is exactly what the budget is for, and watching it work is
the point of the exercise:

```bash
kubectl -n dojo-upgrade-worker get pods -o wide -w
```

Then the kubelet, on worker1:

```bash
sudo apt-mark unhold kubelet kubectl
sudo apt-get install -y kubelet=1.35.8-1.1 kubectl=1.35.8-1.1
sudo apt-mark hold kubelet kubectl
sudo systemctl daemon-reload
sudo systemctl restart kubelet
```

and finally, from the workstation:

```bash
kubectl uncordon worker1
kubectl get nodes
```

## What not to do

If the drain is slow, the temptation is `--force`, `--disable-eviction` or
deleting the PodDisruptionBudget. All three make the command return faster and
all three defeat the purpose — `--disable-eviction` in particular bypasses the
budget entirely and deletes the Pods outright. On a real cluster that is how a
routine node upgrade becomes an outage.

If a drain genuinely cannot proceed, the answer is to find out why the
replacements are not becoming Ready, not to stop asking.

## Checking before you grade

```bash
kubectl get nodes -o wide
ssh worker1 'kubeadm version -o short; apt-mark showhold'
kubectl -n dojo-upgrade-worker get deploy,pdb
```

Both nodes should read 1.35 and `Ready`, neither `SchedulingDisabled`, and
`ledger` back to 3/3.
