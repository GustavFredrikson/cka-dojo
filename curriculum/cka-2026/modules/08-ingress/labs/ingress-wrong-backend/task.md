# The Ingress exists, and every request returns 503

Namespace `dojo-ingress-wrong-backend` serves a site at host
`shop.dojo.test` through Ingress `web`.

Every request comes back `503 Service Temporarily Unavailable`:

```bash
kubectl -n dojo-ingress-wrong-backend exec deploy/client -- \
  wget -T 5 -q -O- --header 'Host: shop.dojo.test' \
  http://ingress-nginx-controller.ingress-nginx.svc.cluster.local/
```

The Deployment is healthy and its Pods are Ready.

Make the request succeed. Leave the host, the path and the Service itself as
they are.
