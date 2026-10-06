# Upgrade the worker without dropping the app

The control plane is on 1.35 and `worker1` is still on 1.34. That skew is
allowed, but it is not where you want to leave a cluster.

Upgrade `worker1` to 1.35.

The `ledger` Deployment in `dojo-upgrade-worker` runs three replicas and has a
PodDisruptionBudget requiring at least two of them available at all times.
Respect it — the budget is there to be honoured, not worked around.

When you are done:

- `kubeadm version` on worker1 reports 1.35
- `kubectl get nodes` shows worker1 with a 1.35 kubelet, `Ready`, and **not**
  cordoned
- all three `ledger` replicas are available again
- the Kubernetes packages on worker1 are held again

The same two obstacles are in your way as on the control plane, and they are
per-node: fixing them on cp1 did nothing here.

**This drill runs once per environment**, for the same reason the last one did.
`dojo reset` will refuse rather than hand you a lab that is already finished.
