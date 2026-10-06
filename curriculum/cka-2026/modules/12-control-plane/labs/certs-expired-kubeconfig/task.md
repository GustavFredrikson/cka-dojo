# An expired client certificate

The `dojo-operator` account has a kubeconfig at `~/ops.kubeconfig` on the
workstation. It is used to read the `ledger` Deployment in namespace
`dojo-certs-expired-kubeconfig`, and it has stopped working:

```
$ kubectl --kubeconfig ~/ops.kubeconfig -n dojo-certs-expired-kubeconfig get deploy
error: You must be logged in to the server (Unauthorized)
```

Nothing else is broken. The cluster is healthy, your own `~/.kube/config` still
works, and the Role and RoleBinding that grant this account its access are
untouched. The client certificate inside `~/ops.kubeconfig` has expired.

Get the account working again, with the same identity and the same access it
had before:

- `kubectl --kubeconfig ~/ops.kubeconfig auth whoami` reports `dojo-operator`
- it can list Deployments in `dojo-certs-expired-kubeconfig`
- it still cannot delete Pods there, read Secrets in `kube-system`, or create
  ClusterRoleBindings

Repair `~/ops.kubeconfig` itself. Handing the account your own administrator
credentials is not a fix, and is checked for. Leave the cluster's own
certificates alone — `kubeadm certs check-expiration` should read exactly the
same afterwards as it does now.

The CA that has to sign the replacement is on `cp1`, and you have root there.
