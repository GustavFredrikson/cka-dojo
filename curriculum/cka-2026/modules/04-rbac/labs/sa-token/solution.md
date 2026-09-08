# Worked solution

## ServiceAccount and RBAC

```bash
kubectl -n dojo-sa-token create serviceaccount reporter

kubectl -n dojo-sa-token create role pod-reader \
  --verb=get,list,watch --resource=pods

kubectl -n dojo-sa-token create rolebinding reporter-can-read \
  --role=pod-reader --serviceaccount=dojo-sa-token:reporter
```

Note the subject syntax. For a ServiceAccount it is
`--serviceaccount=<namespace>:<name>`, and the namespace is not optional even
when it is the one you are already in — the underlying subject is the full
identity `system:serviceaccount:dojo-sa-token:reporter`, and a binding written
without the namespace silently refers to something else.

Check it the way the API server would:

```bash
kubectl -n dojo-sa-token auth can-i list pods \
  --as=system:serviceaccount:dojo-sa-token:reporter
```

## Issuing a token by hand

```bash
kubectl -n dojo-sa-token create token reporter --duration=10m
```

This is the TokenRequest API, and it is worth knowing what changed. Until
Kubernetes 1.24, creating a ServiceAccount also created a `Secret` holding a
token that never expired; those tokens were long-lived credentials sitting in
etcd, and `kubectl get secret` was how you found one. Now:

- Tokens are **requested**, short-lived, and bound to a time and an audience.
- No Secret is created automatically. `kubectl get secrets` in this namespace
  shows nothing.
- A token can be bound to the *object* that uses it, so it stops working when
  that object is deleted.

You can still create a long-lived one deliberately with a
`kubernetes.io/service-account-token` Secret, and you should not want to.

Use it directly to see that authentication and authorisation are separate
things:

```bash
TOKEN=$(kubectl -n dojo-sa-token create token reporter --duration=10m)
kubectl --token="$TOKEN" -n dojo-sa-token get pods
kubectl --token="$TOKEN" -n dojo-sa-token get secrets    # Forbidden
```

The first works, the second is refused — same identity, same token, different
RBAC.

## The projected token

Every Pod already gets a token mounted at
`/var/run/secrets/kubernetes.io/serviceaccount/token`, automatically, with the
API server as its audience. A **projected** token is how you ask for a
different one: a specific audience, a specific lifetime, a specific path.

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: consumer
  namespace: dojo-sa-token
spec:
  serviceAccountName: reporter
  containers:
    - name: consumer
      image: busybox:1.36
      command: ["sleep", "3600"]
      volumeMounts:
        - name: api-token
          mountPath: /var/run/secrets/dojo
          readOnly: true
  volumes:
    - name: api-token
      projected:
        sources:
          - serviceAccountToken:
              audience: dojo-metrics
              expirationSeconds: 3600
              path: token
```

Three fields do the work, and the split between two of them catches people:

- **`path`** is relative to the volume's `mountPath`. `mountPath:
  /var/run/secrets/dojo` plus `path: token` gives
  `/var/run/secrets/dojo/token`. Putting the whole path in `path` creates
  `/var/run/secrets/dojo/var/run/secrets/dojo/token`.
- **`audience`** is who the token is *for*. A service receiving it should
  check that it matches itself; a token minted for `dojo-metrics` is rejected
  by the API server, which expects its own audience. That is the point —
  handing a metrics sidecar a token it could use to call the API server is
  the thing audiences exist to prevent.
- **`expirationSeconds`** is a request, floored at 600 by the API server.
  kubelet refreshes the file at roughly 80% of the lifetime, so a
  long-running process must **re-read the file**, not cache the string.

`serviceAccountName: reporter` is separate from all of this and easy to
forget. Without it the Pod runs as `default` and the projected token names
`default`, so the RBAC you wrote applies to nothing.

## Checking before you grade

```bash
kubectl -n dojo-sa-token get pod consumer -o jsonpath='{.spec.serviceAccountName}'
kubectl -n dojo-sa-token exec consumer -- ls -l /var/run/secrets/dojo/
kubectl -n dojo-sa-token exec consumer -- wc -c /var/run/secrets/dojo/token
```

A file of a few hundred bytes. If you want to see what is in it, its middle
segment is base64 JSON:

```bash
kubectl -n dojo-sa-token exec consumer -- cat /var/run/secrets/dojo/token \
  | cut -d. -f2 | base64 -d 2>/dev/null
```

`aud` is `dojo-metrics`, `sub` is
`system:serviceaccount:dojo-sa-token:reporter`, and `exp` is an hour out.
