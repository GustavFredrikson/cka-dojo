# Worked solution

```bash
kubectl -n dojo-services-follow create deployment web --image=nginx:1.27-alpine --replicas=2
kubectl -n dojo-services-follow rollout status deployment/web
kubectl -n dojo-services-follow expose deployment web --name=web --port=80 --target-port=80
kubectl -n dojo-services-follow get service web
kubectl -n dojo-services-follow get endpointslice -l kubernetes.io/service-name=web
```

The generated Service selector matches the labels on the Deployment's Pod
template, so the EndpointSlice controller publishes both Pod addresses.
