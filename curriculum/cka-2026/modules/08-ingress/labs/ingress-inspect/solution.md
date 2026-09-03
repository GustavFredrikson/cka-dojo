# Explain what an Ingress will and will not match

## Working it out

```bash
kubectl -n dojo-ingress-inspect describe ingress storefront
kubectl -n dojo-ingress-inspect get ingress storefront -o yaml
```

Three facts come straight off that output.

**The host.** `store.dojo.test`. The rule is host-scoped, so a request without
that `Host` header matches nothing and gets a 404 — even from the right
controller, to the right port, with the right path.

**The `/shop` rule is `Prefix`.** It matches `/shop`, `/shop/`, and everything
below it, so `/shop/catalogue` goes to Service `shop`.

**The `/api` rule is `Exact`.** It matches the string `/api` and nothing
else — not `/api/`, not `/api/users`. That is more destructive than it sounds,
because this backend serves a directory:

```bash
C=http://ingress-nginx-controller.ingress-nginx.svc.cluster.local
kubectl -n dojo-ingress-inspect exec deploy/client -- \
  wget -T 5 -S -q -O- --header 'Host: store.dojo.test' $C/api
# HTTP/1.1 301 Moved Permanently
# Location: http://store.dojo.test/api/

kubectl -n dojo-ingress-inspect exec deploy/client -- \
  wget -T 5 -q -O- --header 'Host: store.dojo.test' $C/api/
# 404
```

Read that sequence carefully. `/api` *did* match, reached nginx, and nginx
redirected to `/api/` as it does for any directory. But `/api/` does not match
an `Exact` rule for `/api`, so the redirect the backend just issued lands
nowhere. The rule sabotages its own backend, and no single request shows you
both halves.

Compare with the `Prefix` rule: `/shop`, `/shop/` and `/shop/catalogue` all
match, so the same redirect resolves cleanly.

The two 404s in this exercise come from different places, which is worth
sitting with. `/shop/nonexistent` reaches the backend and the backend has no
such file. `/api/` never matches a rule at all, and nginx answers as the
controller. `kubectl -n ingress-nginx logs deploy/ingress-nginx-controller`
distinguishes them: a request that reached a backend is logged with an
upstream address, one that matched nothing is not.

## Checking before you grade

The exercise changes nothing, so grading only confirms the route still works:

```bash
kubectl -n dojo-ingress-inspect exec deploy/client -- \
  wget -T 5 -q -O- --header 'Host: store.dojo.test' \
  http://ingress-nginx-controller.ingress-nginx.svc.cluster.local/shop/
```
