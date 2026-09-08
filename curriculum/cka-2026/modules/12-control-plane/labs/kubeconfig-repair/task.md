# You cannot reach the cluster

`kubectl` no longer works from the workstation. The cluster itself is healthy
-- the control plane is serving, and both workers are `Ready`.

Repair your access, so that as the `student` user, with no environment
variables set, `kubectl get nodes` lists all three nodes and you still have
full administrative rights.

The fix must live in `~/.kube/config`. Pointing `KUBECONFIG` at a file
somewhere else does not count.
