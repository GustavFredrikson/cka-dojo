# Build a constrained Deployment

In namespace `dojo-workloads-build`, create Deployment `api` with:

- image `nginx:1.27-alpine` and three replicas;
- Pod label `app=api`;
- CPU request `25m` and memory request `32Mi`;
- CPU limit `100m` and memory limit `64Mi`.

Use any valid workflow and leave all three replicas available.
