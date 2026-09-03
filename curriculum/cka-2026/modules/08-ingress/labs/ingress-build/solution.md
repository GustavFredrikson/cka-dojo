# Fan two paths out to two Services

## Working it out

One Ingress can hold many rules, and one host can hold many paths. The whole
job is a single object with a two-entry `paths` list.

The backends here deliberately serve their content *at* `/shop/` and `/api/`,
so no rewriting is involved: the path the client asks for is the path nginx
looks up. In the wild you often need
`nginx.ingress.kubernetes.io/rewrite-target` because the backend expects to be
rooted at `/`; recognising when you do need it starts with knowing what
happens when you do not.

## Fixing it

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata: {name: storefront, namespace: dojo-ingress-build}
spec:
  ingressClassName: nginx
  rules:
    - host: store.dojo.test
      http:
        paths:
          - path: /shop
            pathType: Prefix
            backend:
              service:
                name: shop
                port: {number: 80}
          - path: /api
            pathType: Prefix
            backend:
              service:
                name: api
                port: {number: 80}
```

Or imperatively, which takes repeated `--rule` flags:

```bash
kubectl -n dojo-ingress-build create ingress storefront --class=nginx \
  --rule='store.dojo.test/shop*=shop:80' \
  --rule='store.dojo.test/api*=api:80'
```

The `*` suffix in that shorthand is what produces `pathType: Prefix`; without
it you get `Exact`, and only the bare path would match.

## Checking before you grade

```bash
kubectl -n dojo-ingress-build describe ingress storefront
kubectl -n dojo-ingress-build exec deploy/client -- \
  wget -T 5 -q -O- --header 'Host: store.dojo.test' \
  http://ingress-nginx-controller.ingress-nginx.svc.cluster.local/shop/
```

Each rule should list its Service with a real Pod address in brackets. Both
`/shop/` and `/api/` must return their own marker; if one returns the other's,
the two rules are the wrong way round.
