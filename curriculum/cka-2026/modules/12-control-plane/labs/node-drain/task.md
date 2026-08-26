# Prepare a worker for maintenance

Prepare node `worker1` for operating-system maintenance:

- leave the node `Ready` but unschedulable;
- safely evict ordinary workloads;
- ensure all six replicas of Deployment `web` in namespace `dojo-node-drain`
  are available on other nodes.

DaemonSet Pods may remain. Do not delete the Deployment.
