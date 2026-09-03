# Ingress and Gateway API

## Mental model

```text
client → NodePort → controller Pod → Service → Pod
                         ↑
                    reads Ingress
```

An Ingress object is **configuration, not a proxy**. It is a row in a table
that some controller reads; if no controller claims it, the object is created
happily and nothing whatsoever happens. That is the single most confusing
thing about Ingress, and the reason `ingressClassName` matters.

Gateway API does the same job with the roles split apart:

| Ingress | Gateway API | Owned by |
|---|---|---|
| IngressClass | GatewayClass | the platform |
| — | Gateway (listeners, ports) | the platform |
| Ingress (rules) | HTTPRoute | the app team |

The split is the point: an app team attaches an HTTPRoute to a Gateway it does
not own, and the Gateway decides whether to accept it.

## Objects involved

- `IngressClass` / `GatewayClass`: names a controller. Cluster-scoped.
- `Ingress`: host and path rules pointing at Services, in one namespace.
- `Gateway`: listeners — protocol, port, hostname, and which routes may
  attach.
- `HTTPRoute`: matches and backends, attached to a Gateway via `parentRefs`.
- The controller's own Service: what the outside world actually connects to.

## Commands worth knowing

```bash
kubectl get ingressclass
kubectl describe ingress NAME -n NAMESPACE
kubectl get ingress -A
kubectl -n ingress-nginx get svc ingress-nginx-controller
kubectl get gatewayclass
kubectl describe gateway NAME -n NAMESPACE      # listener status
kubectl describe httproute NAME -n NAMESPACE    # parent acceptance
kubectl -n ingress-nginx logs deploy/ingress-nginx-controller
```

## Diagnostic workflow

```text
A request from outside does not arrive
  ↓ does the Ingress exist and name a class that exists?
no  → set ingressClassName; check `kubectl get ingressclass`
yes ↓ kubectl describe ingress -- is a backend listed and non-empty?
no  → the Service name or port in the rule is wrong
yes ↓ does that Service have endpoints?
no  → ordinary Service debugging; the Ingress is innocent
yes ↓ what does the controller answer?
404 → no rule matched: host header, path, or pathType
503 → a rule matched but the backend is unreachable
```

For Gateway API the same walk reads: is the Gateway `Programmed`, and does the
HTTPRoute show `Accepted` under its parent? A route whose `parentRefs` do not
match a listener is simply ignored.

## Common CKA failure modes

- Creating an Ingress with no `ingressClassName` on a cluster whose controller
  is not the default: nothing serves it, and nothing complains.
- `pathType: Exact` where `Prefix` was meant, so only the bare path matches.
- Naming the Service port by number when the Service names it, or vice versa.
- Expecting the Ingress to work without a `Host` header, when the rule
  specifies a host.
- Reading a 503 as an Ingress problem when the backing Service has no
  endpoints.
- An HTTPRoute in a namespace the Gateway's listener does not allow, which
  fails as "not accepted" rather than as an error on the route.

## 5-minute walkthrough

Run `kubectl get ingressclass` and `kubectl get gatewayclass` to see which
controllers this cluster has. Then look at how the outside world reaches them:
`kubectl -n ingress-nginx get svc ingress-nginx-controller` shows the NodePort
that the exercises curl. Finally run `kubectl explain ingress.spec.rules.http.paths`
and note that `pathType` is required.

## Labs

```bash
dojo start ingress-follow
dojo start ingress-build
dojo start ingress-inspect
dojo start ingress-wrong-backend
dojo start gateway-build
```
