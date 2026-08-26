# Worked solution

```bash
kubectl -n dojo-rbac-guided edit rolebinding builder-pods
kubectl auth can-i list pods -n dojo-rbac-guided \
  --as=system:serviceaccount:dojo-rbac-guided:builder
```

Change only the subject name from `wrong-subject` to `builder`.
