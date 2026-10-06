# This account cannot do anything

The `dojo-billing` account reads the `invoices` Deployment in namespace
`dojo-certs-identity-refused`. Its kubeconfig is at `~/billing.kubeconfig` on
the workstation, and every command it runs is refused.

Every other account is fine. Your own `~/.kube/config` works, the cluster is
healthy, and nothing was changed on purpose except what belongs to this account.

Restore it. When you are done:

- `kubectl --kubeconfig ~/billing.kubeconfig auth whoami` reports `dojo-billing`
- it can list Deployments in `dojo-certs-identity-refused`
- it still cannot delete them, read Secrets in `kube-system`, or create
  ClusterRoleBindings

Do not widen what this account is allowed to do, and do not hand it your own
credentials. Both are checked.
