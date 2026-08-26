# RBAC

## Mental model

Kubernetes RBAC answers one question:

```text
May subject S perform verb V on resource R in scope N?
```

Permissions and recipients are separate objects:

```text
User / Group / ServiceAccount
          ↓ subject
RoleBinding or ClusterRoleBinding
          ↓ roleRef
Role or ClusterRole
          ↓ rules
API groups + resources + verbs
```

A Role describes permissions inside one namespace. A ClusterRole describes a
reusable set of permissions and can also include cluster-scoped resources. A
binding determines who receives those permissions and, critically, where.

## Objects involved

- `Role`: namespaced rules.
- `ClusterRole`: cluster-scoped rule object; may be referenced by either type
  of binding.
- `RoleBinding`: grants its referenced role inside one namespace only.
- `ClusterRoleBinding`: grants its referenced ClusterRole cluster-wide.
- `ServiceAccount`: a namespaced workload identity.
- Users and groups: names presented by authentication; Kubernetes has no User
  API object to create.

A RoleBinding may reference a ClusterRole. The permissions still apply only in
the RoleBinding's namespace. This is useful for defining one reusable reader
role and binding it in selected namespaces.

## Commands worth knowing

```bash
kubectl create role -h
kubectl create rolebinding -h
kubectl auth can-i get pods -n backend --as alice
kubectl auth can-i --list -n backend --as alice
kubectl get role,rolebinding -n backend
kubectl describe role -n backend namespace-reader
kubectl describe rolebinding -n backend alice-reader
```

For a ServiceAccount:

```bash
kubectl auth can-i list pods -n backend \
  --as system:serviceaccount:backend:builder
```

Resource API groups matter. Pods belong to the core group `""`; Deployments
belong to `apps`:

```yaml
rules:
  - apiGroups: [""]
    resources: ["pods"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["apps"]
    resources: ["deployments"]
    verbs: ["get", "list", "watch"]
```

## Diagnostic workflow

Turn the task into explicit authorization checks before writing YAML.

```text
Who is the subject?
  user, group, or service account
        ↓
Which exact verbs and resources?
        ↓
Namespaced or cluster-wide scope?
        ↓
Role/RoleBinding or ClusterRole/ClusterRoleBinding
        ↓
Ask the API server with kubectl auth can-i
        ↓
Check required yes answers and forbidden no answers
```

If access is denied, inspect both halves:

1. Does the binding name the correct subject and namespace?
2. Does `roleRef` point at the intended role kind and name?
3. Do the role rules use the correct API group, resource and verb?

The API server's answer is the final truth. Reading YAML and deciding that it
“looks right” is not verification.

## Common CKA failure modes

- Using a ClusterRoleBinding when access should be limited to one namespace.
- Binding a role to the wrong subject kind or ServiceAccount namespace.
- Omitting `list` or `watch` when “read access” requires all three read verbs.
- Writing `apiGroups: [""]` for Deployments instead of `apps`.
- Granting `secrets` through a broad wildcard when the task asks only for Pods.
- Editing `roleRef`: it is immutable, so a binding with the wrong reference
  must be recreated.
- Testing as the current cluster administrator instead of impersonating the
  intended subject with `--as`.

## 5-minute walkthrough

Create a namespaced reader and test it:

```bash
kubectl create namespace demo-rbac
kubectl -n demo-rbac create role pod-reader \
  --verb=get,list,watch --resource=pods
kubectl -n demo-rbac create rolebinding alice-pod-reader \
  --role=pod-reader --user=alice

kubectl auth can-i list pods -n demo-rbac --as alice
kubectl auth can-i delete pods -n demo-rbac --as alice
kubectl auth can-i list pods -n default --as alice
```

The expected answers are `yes`, `no`, `no`. Remove it afterwards:

```bash
kubectl delete namespace demo-rbac
```

## Labs

The path progresses from an explicit ServiceAccount grant to independent
construction, inspection, known repair and symptom-based authorization repair.

```bash
dojo start rbac-follow
dojo start rbac-namespace-reader
dojo start rbac-inspect
dojo start rbac-guided-binding
dojo start rbac-contextual
```
