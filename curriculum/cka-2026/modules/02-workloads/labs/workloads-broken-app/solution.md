# Worked solution

## Working it out

```bash
kubectl -n dojo-workloads-broken get deploy,rs,pod
kubectl -n dojo-workloads-broken describe pod -l app=web
kubectl -n dojo-workloads-broken logs deployment/web --previous
kubectl -n dojo-workloads-broken get deploy web -o yaml
```

`ImagePullBackOff` points to the image reference, a repeatedly terminated
container points to command/logs, and `CreateContainerConfigError` points to a
referenced ConfigMap or Secret. Correct only the field responsible, then watch
`kubectl rollout status deployment/web`.
