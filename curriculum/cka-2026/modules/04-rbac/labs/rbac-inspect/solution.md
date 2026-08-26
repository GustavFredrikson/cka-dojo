# Worked solution

```bash
kubectl -n dojo-rbac-inspect get serviceaccount,role,rolebinding
kubectl -n dojo-rbac-inspect describe rolebinding viewer-config
kubectl -n dojo-rbac-inspect describe role config-reader
kubectl auth can-i --list -n dojo-rbac-inspect --as=system:serviceaccount:dojo-rbac-inspect:viewer
```
