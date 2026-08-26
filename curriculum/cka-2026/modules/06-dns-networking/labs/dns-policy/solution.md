# Worked solution

```bash
kubectl -n dojo-dns-policy patch deployment dnscheck --type merge -p '{"spec":{"template":{"spec":{"dnsPolicy":"ClusterFirst"}}}}'
kubectl -n dojo-dns-policy rollout status deployment/dnscheck
```
