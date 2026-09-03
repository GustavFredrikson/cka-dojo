# Route your first request from outside

Namespace `dojo-ingress-follow` runs Deployment `web` behind Service `web` on
port `80`, plus a `client` Pod to send requests from.

This cluster has no external load balancer, so you reach the Ingress
controller through its own Service inside the cluster:

```text
http://ingress-nginx-controller.ingress-nginx.svc.cluster.local
```

The controller decides what to do with a request by looking at its `Host`
header, which is why every command below sets one.

Work through the checkpoints with `dojo check`.
