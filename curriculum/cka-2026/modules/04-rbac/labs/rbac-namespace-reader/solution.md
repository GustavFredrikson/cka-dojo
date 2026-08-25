# Solution

## Working it out

Two decisions:

1. **Namespaced or cluster-wide?** The access must apply to `backend` only, so
   the pair is `Role` + `RoleBinding`. `ClusterRole` + `ClusterRoleBinding`
   would grant the permission in every namespace.
2. **Which resources?** Pods are in the core (`""`) group. Deployments are in
   the `apps` group, and a Role has to say so.

Note that a `RoleBinding` may reference a `ClusterRole`; that also scopes the
permission to one namespace and is a perfectly good answer, as long as the
ClusterRole grants no more than the read verbs asked for.

## Fixing it

Imperatively:

```
kubectl -n backend create role namespace-reader \
  --verb=get,list,watch \
  --resource=pods,deployments.apps

kubectl -n backend create rolebinding alice-namespace-reader \
  --role=namespace-reader \
  --user=alice
```

Or declaratively:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: namespace-reader
  namespace: backend
rules:
  - apiGroups: [""]
    resources: ["pods"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["apps"]
    resources: ["deployments"]
    verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: alice-namespace-reader
  namespace: backend
subjects:
  - kind: User
    name: alice
    apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: Role
  name: namespace-reader
  apiGroup: rbac.authorization.k8s.io
```

## Checking before you grade

Ask the API server rather than reading your own YAML back:

```
kubectl auth can-i list pods        -n backend     --as alice   # yes
kubectl auth can-i get deployments  -n backend     --as alice   # yes
kubectl auth can-i delete pods      -n backend     --as alice   # no
kubectl auth can-i get secrets      -n backend     --as alice   # no
kubectl auth can-i get pods         -n default     --as alice   # no
```

`kubectl auth can-i --list -n backend --as alice` shows everything at once.
