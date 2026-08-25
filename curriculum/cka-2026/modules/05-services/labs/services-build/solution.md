# Worked solution

One efficient solution is:

```bash
kubectl -n dojo-services-build expose deployment api --name=api --port=8080 --target-port=http
kubectl -n dojo-services-build get svc api
kubectl -n dojo-services-build get endpointslice -l kubernetes.io/service-name=api
```

The Deployment's Pods carry `app: api`, and `kubectl expose deployment`
derives that selector. The named `targetPort` resolves to the container's
named port `http`.
