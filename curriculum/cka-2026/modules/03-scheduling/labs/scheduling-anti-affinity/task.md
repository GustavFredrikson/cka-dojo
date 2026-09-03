# Keep two replicas off the same node

A single node failure must never take out both replicas of this service.

In namespace `dojo-scheduling-anti-affinity`, create Deployment `web`:

- image `nginx:1.27-alpine`, `2` replicas
- Pods labelled `app=web`
- the scheduler must **refuse** to place two of these Pods on the same node,
  rather than merely preferring to spread them

Both replicas must be running, on different nodes.
