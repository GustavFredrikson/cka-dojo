# Worked solution

```bash
kubectl -n dojo-dns-frontend edit deployment frontend
# Use http://api.dojo-dns-backend.svc.cluster.local/
kubectl -n dojo-dns-frontend rollout status deployment/frontend
```
