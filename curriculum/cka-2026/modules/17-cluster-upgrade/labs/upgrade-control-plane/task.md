# Upgrade the control plane to 1.35

This cluster is on Kubernetes 1.34. Take `cp1` — and only `cp1` — to 1.35.

When you are done:

- `kubeadm version` on cp1 reports 1.35
- the API server, controller-manager and scheduler are running 1.35 images
- `kubectl get nodes` shows cp1 with a 1.35 kubelet, `Ready`, and **not**
  cordoned
- the `ledger` Deployment in `dojo-upgrade-control-plane` never stops being
  available
- the Kubernetes packages are held again when you finish, the way you found them

`worker1` stays on 1.34. That is allowed — a kubelet may lag the API server,
never lead it — and upgrading it is the next exercise.

Two things are in your way on purpose, and both are in your way on the real
exam: the Kubernetes packages are pinned with `apt-mark hold`, and the apt
repository on this node carries only the 1.34 minor.

**This drill runs once per environment.** A kubeadm upgrade is one-way, so
`dojo reset` cannot put the cluster back on 1.34 — it will tell you so and
refuse. To run it again, rebuild first with
`dojo env reset --profile upgrade-1.34`.
