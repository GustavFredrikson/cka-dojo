# Solution

## Working it out

A Service sends traffic to the addresses listed in its EndpointSlices. Those
are populated by matching the Service's `spec.selector` against Pod labels, and
the traffic is delivered to `spec.ports[].targetPort` on the Pod.

So there are exactly two joints that can be wrong, and one command tells you
which:

```
kubectl -n shop get endpointslice -l kubernetes.io/service-name=web -o yaml
```

- **No addresses at all** -> the selector does not match any Pod. Compare
  `kubectl -n shop get svc web -o jsonpath='{.spec.selector}'` with
  `kubectl -n shop get pods --show-labels`.
- **Addresses present, but requests fail** -> the selector is fine and the
  targetPort is wrong. Compare the Service's `targetPort` with the container's
  declared port:
  `kubectl -n shop get deploy web -o jsonpath='{.spec.template.spec.containers[0].ports}'`.

The baseline names the container port `http`, so a correct Service uses either
`targetPort: http` or `targetPort: 80`.

## Fixing it

Any of these is equally correct; grading looks at the resulting state.

```
kubectl -n shop edit svc web
```

or, for a wrong selector:

```
kubectl -n shop patch svc web -p '{"spec":{"selector":{"app":"web"}}}'
```

or, for a wrong target port:

```
kubectl -n shop patch svc web -p '{"spec":{"ports":[{"name":"http","port":80,"targetPort":"http","protocol":"TCP"}]}}'
```

## Checking before you grade

```
kubectl -n shop get endpointslice -l kubernetes.io/service-name=web
kubectl -n shop run probe --rm -it --image=busybox:1.36 --restart=Never -- wget -qO- http://web
```
