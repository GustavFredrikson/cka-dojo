# Give the cluster a pod network

`cp1` is a working control plane — `kubeadm init` has run, the static Pods are
up, and your workstation kubeconfig points at it. But:

```
$ kubectl get nodes
NAME   STATUS     ROLES           AGE   VERSION
cp1    NotReady   control-plane   2m    v1.35.8

$ kubectl -n kube-system get pods
NAME                     READY   STATUS    RESTARTS   AGE
coredns-...              0/1     Pending   0          2m
coredns-...              0/1     Pending   0          2m
```

Nothing is misconfigured. Kubernetes does not ship a pod network, and this
cluster does not have one yet.

Install one, so that `cp1` becomes `Ready` and CoreDNS starts.

The cluster was built with `--pod-network-cidr=10.244.0.0/16`, and whatever you
install has to use that range — a pod network on a different range will look
like it worked and leave the cluster subtly broken. Grading checks the address
CoreDNS actually gets.

Any CNI plugin is accepted.

**Note:** this lab builds a cluster during setup, so `dojo start` and
`dojo reset` each take three to five minutes.
