# Worked solution

```bash
kubectl -n dojo-services-inspect get pods --show-labels
kubectl -n dojo-services-inspect get svc web -o yaml
kubectl -n dojo-services-inspect get endpointslice -l kubernetes.io/service-name=web -w
kubectl -n dojo-services-inspect patch svc web -p '{"spec":{"selector":{"app":"missing"}}}'
kubectl -n dojo-services-inspect patch svc web -p '{"spec":{"selector":{"app":"web"}}}'
```

The important relationship is `app=web`. A Service does not maintain a fixed
list of Pods; controllers continually derive EndpointSlices from its selector.
