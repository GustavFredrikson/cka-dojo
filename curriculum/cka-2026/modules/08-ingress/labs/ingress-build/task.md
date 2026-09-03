# Fan two paths out to two Services

Namespace `dojo-ingress-build` runs two independent applications, each behind
its own Service on port `80`:

- `shop`, which serves content under `/shop/`
- `api`, which serves content under `/api/`

Create a **single** Ingress named `storefront`, served by class `nginx`, for
host `store.dojo.test`, so that:

- `/shop/` and anything beneath it reaches Service `shop`
- `/api/` and anything beneath it reaches Service `api`

Send test requests from the `client` Pod against
`http://ingress-nginx-controller.ingress-nginx.svc.cluster.local`, remembering
the `Host` header.
