# Worked solution

```bash
kubectl -n dojo-netpol-guided patch networkpolicy allow-client --type=merge \
  -p='{"spec":{"ingress":[{"from":[{"podSelector":{"matchLabels":{"role":"client"}}}],"ports":[{"protocol":"TCP","port":80}]}]}}'
```

Retest from `deployment/client`.
