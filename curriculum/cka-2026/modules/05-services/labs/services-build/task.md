# Expose a workload without a recipe

Deployment `api` is healthy in namespace `dojo-services-build`.

Create a ClusterIP Service named `api` that listens on port `8080` and sends
traffic to the workload's HTTP port. Do not change the Deployment. You choose
the commands or manifests.
