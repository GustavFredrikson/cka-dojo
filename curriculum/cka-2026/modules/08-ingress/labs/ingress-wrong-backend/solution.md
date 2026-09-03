# The Ingress exists, and every request returns 503

## Working it out

Start by reading the status code as evidence. The controller answered, so the
request got all the way to nginx and a rule matched — otherwise you would have
seen 404. A 503 means the rule matched and nginx had nowhere to send it.

That narrows things to the gap between the rule and the Service.

```bash
kubectl -n dojo-ingress-wrong-backend describe ingress web
```

The `Rules` block resolves each backend, and this one shows the Service with
no endpoints behind it. Now check whether that is the Service's fault:

```bash
kubectl -n dojo-ingress-wrong-backend get svc web
kubectl -n dojo-ingress-wrong-backend get endpointslice -l kubernetes.io/service-name=web
kubectl -n dojo-ingress-wrong-backend get pods -l app=web
```

The Service has a healthy endpoint on port 80, and the Pod is Ready. So the
Service is fine and the Ingress is asking it for something it does not offer:

```bash
kubectl -n dojo-ingress-wrong-backend get ingress web \
  -o jsonpath='{.spec.rules[0].http.paths[0].backend.service}'; echo
kubectl -n dojo-ingress-wrong-backend get svc web -o jsonpath='{.spec.ports}'; echo
```

The rule asks for port `8080`. The Service publishes `80`, named `http`. A
backend port that does not exist on the Service resolves to nothing, and nginx
returns 503 for an empty upstream.

Note what this is *not*: the Pod's `containerPort` is irrelevant here. The
Ingress addresses the **Service** port, and the Service maps that to the
container. Chasing the container port is the usual wrong turn.

## Fixing it

Point the rule at a port the Service actually publishes — by number:

```bash
kubectl -n dojo-ingress-wrong-backend patch ingress web --type=json \
  -p '[{"op":"replace","path":"/spec/rules/0/http/paths/0/backend/service/port","value":{"number":80}}]'
```

or by name, which is the more durable choice because it survives a port
renumbering:

```yaml
            backend:
              service:
                name: web
                port:
                  name: http
```

`kubectl edit ingress web` does either. The controller reloads within a second
or two.

## Checking before you grade

```bash
kubectl -n dojo-ingress-wrong-backend describe ingress web
kubectl -n dojo-ingress-wrong-backend exec deploy/client -- \
  wget -T 5 -q -O- --header 'Host: shop.dojo.test' \
  http://ingress-nginx-controller.ingress-nginx.svc.cluster.local/
```

The backend line should now carry a Pod address, and the request should return
`served-by-web`.
