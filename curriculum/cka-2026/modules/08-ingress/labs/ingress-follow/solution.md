# Route your first request from outside

## Working it out

Three things have to line up before a request arrives:

1. a controller is running and owns a class — `kubectl get ingressclass`
2. an Ingress names that class and a host
3. the rule points at a Service that has endpoints

The 404 in the second checkpoint is the controller answering. That matters:
the request reached nginx, and nginx had no rule for `shop.dojo.test`. A
connection refused or a timeout would have meant something entirely
different — that you never got to the controller at all.

## Fixing it

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata: {name: web, namespace: dojo-ingress-follow}
spec:
  ingressClassName: nginx
  rules:
    - host: shop.dojo.test
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: web
                port:
                  number: 80
```

```bash
kubectl apply -f ingress.yaml
```

The nesting under `backend` is the part worth memorising — `service.name` and
`service.port.number` are two levels deeper than feels natural, and
`kubectl explain ingress.spec.rules.http.paths.backend` is the fastest way to
recover it under pressure.

`pathType` is required. `Prefix` matches `/` and everything under it; `Exact`
would match only the bare `/`.

There is an imperative form, which is quicker when the shape is simple:

```bash
kubectl -n dojo-ingress-follow create ingress web \
  --class=nginx --rule='shop.dojo.test/*=web:80'
```

Give the controller a couple of seconds to reload, then:

```bash
kubectl -n dojo-ingress-follow exec deploy/client -- \
  wget -T 5 -q -O- --header 'Host: shop.dojo.test' \
  http://ingress-nginx-controller.ingress-nginx.svc.cluster.local/
```

## Checking before you grade

```bash
kubectl -n dojo-ingress-follow describe ingress web
```

Under `Rules` you should see the host, the path, and `web:80` with the Pod
address in brackets. An empty backend list, or `<error: endpoints not found>`,
means the rule is pointing at something that does not exist.

Dropping the `Host` header gives you a 404 again — the rule is host-scoped,
and that is the single most common reason a working Ingress looks broken from
a browser typing an IP address.
