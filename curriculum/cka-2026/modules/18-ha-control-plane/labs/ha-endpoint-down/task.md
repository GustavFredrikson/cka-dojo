# Nobody can reach the cluster

From the workstation:

```
$ kubectl get nodes
The connection to the server k8s-api:6443 was refused - did you specify the
right host or port?
```

From `cp1`, `cp2` or `cp3`, with their own admin kubeconfigs, everything works
normally. The nodes are Ready, the workloads are running, etcd is healthy.

Get `kubectl` working from the workstation again.

Leave the kubeconfig pointing where it points. Repointing it at one of the
control planes would make the command work and would undo the reason this
cluster has three of them — it is checked.
