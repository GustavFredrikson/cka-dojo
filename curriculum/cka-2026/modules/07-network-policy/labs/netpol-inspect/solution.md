# Worked solution

```bash
kubectl -n dojo-netpol-inspect get pods --show-labels
kubectl -n dojo-netpol-inspect describe networkpolicy allow-client
kubectl -n dojo-netpol-inspect get networkpolicy allow-client -o yaml
```

The path is `role=client` → `app=server` on `TCP/80`.
