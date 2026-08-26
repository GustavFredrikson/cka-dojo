# Worked solution

```bash
kubectl -n dojo-rbac-contextual get role,rolebinding -o yaml
kubectl auth can-i list deployments -n dojo-rbac-contextual \
  --as=system:serviceaccount:dojo-rbac-contextual:deploy-reader
```

For `wrong-api-group`, change the Deployment rule to `apiGroups: [apps]`.
For `wrong-role-ref`, recreate the binding so it references `deployment-reader`.
Verify the required `yes` answers and forbidden `no` answers.
