# Read what the cluster is actually using

Namespace `dojo-observability-top` runs two Deployments, `grinder` and `dozer`.
One of them reserves far more CPU than it uses; the other does the opposite.

Use the metrics API and the node's own accounting to answer each checkpoint,
then run `dojo check <answer>`.

metrics-server samples on an interval. If `kubectl top` reports that metrics
are not available yet, wait about a minute and try again.
