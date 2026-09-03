# Do the same job with the Gateway API

Namespace `dojo-gateway-build` runs Deployment `web` behind Service `web` on
port `80`.

Expose it through the Gateway API rather than through an Ingress. The cluster
already provides a GatewayClass — find it with `kubectl get gatewayclass`.

1. Create Gateway `storefront` in that namespace, using that class, with a
   single HTTP listener named `http` on port `80` that accepts routes from its
   own namespace.
2. Create HTTPRoute `web` in the same namespace, attached to that Gateway,
   matching hostname `gw.dojo.test` and path prefix `/`, with Service `web`
   port `80` as its backend.

The Gateway must end up reporting `Programmed`, and the HTTPRoute must be
`Accepted` by it.
