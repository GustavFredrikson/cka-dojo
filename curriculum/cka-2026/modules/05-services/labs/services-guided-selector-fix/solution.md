# Worked solution

```bash
kubectl -n dojo-services-guided get pods --show-labels
kubectl -n dojo-services-guided get svc web -o yaml
kubectl -n dojo-services-guided patch svc web -p '{"spec":{"selector":{"app":"web"}}}'
kubectl -n dojo-services-guided get endpointslice -l kubernetes.io/service-name=web
```

The repair restores the selector-to-label match. EndpointSlice reconciliation
then restores the Pod addresses without recreating the Service.
