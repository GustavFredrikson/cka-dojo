# Cap what a namespace may consume

Namespace `dojo-quota-build` is being handed to a team that must not be able
to fill the cluster.

1. Create ResourceQuota `team-quota` in that namespace allowing at most:
   - `4` pods
   - `1` CPU and `1Gi` of memory in total *requests*
   - `2` CPU and `2Gi` of memory in total *limits*

2. Then deploy inside those bounds: Deployment `web`, image
   `nginx:1.27-alpine`, `2` replicas, each container requesting `100m` CPU and
   `64Mi` memory with limits of `200m` CPU and `128Mi` memory.

Both replicas must end up running, and the quota must show them counted
against it.
