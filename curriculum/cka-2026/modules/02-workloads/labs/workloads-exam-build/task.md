# Build a constrained production workload

In namespace `dojo-workloads-exam`, create Deployment `processor` with:

- 3 replicas using `nginx:1.27-alpine`;
- Pod label `app=processor`;
- CPU request `100m` and limit `250m`;
- memory request `64Mi` and limit `128Mi`;
- required node affinity placing Pods on node `worker2` using the standard
  hostname label.
