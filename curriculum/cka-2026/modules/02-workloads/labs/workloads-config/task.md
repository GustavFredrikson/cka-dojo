# Configure a workload from a ConfigMap and Secret

In namespace `dojo-workloads-config`:

1. Create ConfigMap `web-settings` containing `APP_MODE=production`.
2. Create Secret `api-credentials` containing `token=swordfish`.
3. Create Deployment `web`, two replicas, image `nginx:1.27-alpine`.
4. Import all keys from both configuration objects into the containers using
   `envFrom`.
