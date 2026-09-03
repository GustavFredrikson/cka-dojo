# Do the same job with the Gateway API

## Working it out

Gateway API splits what Ingress packs into one object:

- the **Gateway** owns the listeners — protocol, port, and which routes may
  attach to it. In a real cluster the platform team owns this.
- the **HTTPRoute** owns matching and backends. The app team owns this, and
  attaches it to a Gateway through `parentRefs`.

So there are two objects and two status conditions to satisfy: the Gateway
becomes `Programmed` when the controller has actually configured a data plane
for it, and the HTTPRoute becomes `Accepted` when its parent Gateway agrees to
take it.

Find the class first — never assume the name:

```bash
kubectl get gatewayclass
```

## Fixing it

```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata: {name: storefront, namespace: dojo-gateway-build}
spec:
  gatewayClassName: nginx
  listeners:
    - name: http
      protocol: HTTP
      port: 80
      allowedRoutes:
        namespaces: {from: Same}
---
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata: {name: web, namespace: dojo-gateway-build}
spec:
  parentRefs:
    - name: storefront
  hostnames: ["gw.dojo.test"]
  rules:
    - matches:
        - path: {type: PathPrefix, value: /}
      backendRefs:
        - name: web
          port: 80
```

Two things catch people out. `backendRefs` takes the **port number** directly,
with no nested `service:` block — it is flatter than the Ingress equivalent,
not deeper. And the path match type is `PathPrefix`, not `Prefix`; the Ingress
spelling is rejected.

There is no `kubectl create gateway`; these are CRDs, so they are written by
hand.

## Checking before you grade

```bash
kubectl -n dojo-gateway-build describe gateway storefront
kubectl -n dojo-gateway-build describe httproute web
```

The Gateway should show `Programmed=True`, and the route should list the
Gateway under `Parents` with `Accepted=True`. A route that reports
`NotAllowedByListeners` is in a namespace the listener does not accept; one
that reports `NoMatchingParent` is naming a Gateway or listener that does not
exist.

The controller provisions an nginx data plane per Gateway, so a new Deployment
and Service appear in the namespace once the Gateway is programmed:

```bash
kubectl -n dojo-gateway-build get deploy,svc
```
