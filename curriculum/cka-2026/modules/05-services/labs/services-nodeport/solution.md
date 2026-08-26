# Worked solution

```bash
kubectl -n dojo-services-nodeport expose deployment web --name web-node --type NodePort --port 80 --target-port 80
kubectl -n dojo-services-nodeport patch service web-node -p '{"spec":{"ports":[{"port":80,"targetPort":80,"nodePort":30080}]}}'
```
