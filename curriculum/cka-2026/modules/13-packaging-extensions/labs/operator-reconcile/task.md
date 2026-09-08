# A change that will not stay changed

This cluster's CNI is managed by an operator.

Future rollouts of the `calico-node` DaemonSet should be allowed to take two
nodes out at a time instead of one — that is,
`spec.updateStrategy.rollingUpdate.maxUnavailable` should be `2`.

Someone already tried editing the DaemonSet directly. It went back to `1`
within half a minute.

Work out why, and make the change in the place that will hold. The CNI must
stay healthy on all three nodes throughout.

Answer each checkpoint with `dojo check <answer>`.
