# Worked solution

```bash
kubectl -n dojo-services-port patch service api -p '{"spec":{"ports":[{"port":8080,"targetPort":80}]}}'
```
