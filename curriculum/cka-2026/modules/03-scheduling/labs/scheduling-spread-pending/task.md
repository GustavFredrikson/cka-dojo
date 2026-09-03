# No replica will schedule anywhere

Deployment `checkout` in namespace `dojo-scheduling-spread-pending` reports
`0/3` available. Every one of its Pods is `Pending`, and they stay that way —
this is not a transient wait.

All three nodes are Ready, and the cluster has ample CPU and memory free. The
`cache` Deployment in the same namespace is healthy and must stay that way.

Bring `checkout` to three available replicas. Its Pod template should still
express *some* placement preference rather than none at all — dropping the
requirement entirely is not the answer.
