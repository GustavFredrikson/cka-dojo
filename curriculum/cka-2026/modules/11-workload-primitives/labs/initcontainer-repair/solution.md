# Solution

## Working it out

```bash
kubectl -n dojo-initcontainer get pods
```

```
NAME                   READY   STATUS     RESTARTS   AGE
web-7d4c...-2xk9n      0/1     Init:0/1   0          4m
```

`Init:0/1` is not an error. It means: zero of one init containers have
completed, so no app container has been allowed to start. Init containers run
to completion, in order, before anything else -- so a Pod in this state is
*waiting*, and the only question worth asking is what for.

The instinct is `kubectl logs`, and it fails:

```bash
kubectl -n dojo-initcontainer logs web-7d4c...-2xk9n
Defaulted container "web" out of: web, wait-for-db (init)
Error from server (BadRequest): container "web" in pod "..." is waiting to start: PodInitializing
```

That message is about the wrong container. `kubectl logs` defaults to the
first *app* container, which by definition has not started. Read the first
line, though -- it lists every container in the Pod and marks which are init
containers, so the failed command hands you the name you needed. Use it:

```bash
kubectl -n dojo-initcontainer logs web-7d4c...-2xk9n -c wait-for-db
```

```
waiting for db
** server can't find db.dojo-initcontainer.svc.cluster.local: NXDOMAIN
waiting for db
```

`describe` gets you to the same place from the other direction -- it lists
init containers separately from containers, with their own state:

```bash
kubectl -n dojo-initcontainer describe pod web-7d4c...-2xk9n | head -30
```

So: the Pod is gated on a DNS name that does not resolve, and the name is a
Service in its own namespace.

## Fixing it

Create the Service. That is all -- the task forbids touching the Pod template,
and it should: an init container that waits for a dependency is doing its job.
The dependency is what is missing.

```bash
kubectl -n dojo-initcontainer create service clusterip db --tcp=5432:5432
```

The init container's loop exits within a couple of seconds and the rollout
proceeds.

The reason this is enough is worth holding onto: **a ClusterIP Service gets a
DNS A record when it is created, not when it has endpoints.** There is no
Deployment behind `db` and there does not need to be -- `nslookup` succeeds
against the Service's cluster IP regardless. If the init container had probed
the *port* instead of the name, you would have needed a real backend.

Two other routes, both valid:

```bash
# A headless Service instead -- also gets a record, of a different kind.
kubectl -n dojo-initcontainer create service clusterip db --clusterip=None
```

or write the Service as YAML with a selector matching a workload you also
create. Grading asks for a Service named `db` and an available Deployment; it
does not care which shape you chose.

What is *not* a fix: removing the init container, or overriding its command to
`true`. Both make the symptom go away and leave the app starting before its
database is reachable -- which is the bug the init container exists to
prevent. Grading checks the init container is still named `wait-for-db` for
that reason.

## Checking before you grade

```bash
kubectl -n dojo-initcontainer get svc db
kubectl -n dojo-initcontainer rollout status deployment/web
kubectl -n dojo-initcontainer get pods
```

`2/2` available, and `READY 1/1` on both Pods.
