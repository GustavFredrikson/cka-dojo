# Configure CPU autoscaling

In namespace `dojo-workloads-hpa`, configure Deployment `web` to autoscale
between **2 and 6 replicas**, targeting **60% average CPU utilization**.

Do not replace the Deployment.
