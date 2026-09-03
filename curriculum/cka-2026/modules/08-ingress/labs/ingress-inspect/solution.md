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

**The `/api` rule is `Exact`.** It matches `/api` and nothing else. `/api/`
and `/api/users` do not match it, fall through to no rule at all, and come
back 404. This is the difference the exercise is built around, and it is a
common real-world outage: someone writes `Exact` meaning "this path", and
every sub-path silently stops working.

Prove it rather than trusting the YAML:

```bash
C=http://ingress-nginx-controller.ingress-nginx.svc.cluster.local
kubectl -n dojo-ingress-inspect exec deploy/client -- \
  wget -T 5 -q -O- --header 'Host: store.dojo.test' $C/shop/catalogue   # 404 from the shop Pod, not the controller
kubectl -n dojo-ingress-inspect exec deploy/client -- \
  wget -T 5 -q -O- --header 'Host: store.dojo.test' $C/api/users        # 404 from the controller
```

Those two 404s come from different places, which is worth sitting with. The
first reached the backend and the backend had no such file. The second never
matched a rule.

## Checking before you grade

The exercise changes nothing, so grading only confirms the route still works:

```bash
kubectl -n dojo-ingress-inspect exec deploy/client -- \
  wget -T 5 -q -O- --header 'Host: store.dojo.test' \
  http://ingress-nginx-controller.ingress-nginx.svc.cluster.local/shop/
```
