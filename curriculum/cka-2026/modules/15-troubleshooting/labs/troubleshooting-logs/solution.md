# Worked solution

```bash
kubectl -n dojo-troubleshooting-logs get pods
kubectl -n dojo-troubleshooting-logs logs deployment/collector -c reporter --previous
kubectl -n dojo-troubleshooting-logs edit deployment collector
# reporter command: [sh, -c, 'while true; do echo healthy; sleep 10; done']
kubectl -n dojo-troubleshooting-logs rollout status deployment/collector
```
