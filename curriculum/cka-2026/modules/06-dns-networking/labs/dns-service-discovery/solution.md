# Worked solution

```bash
kubectl -n dojo-dns-service expose deployment api --name api --port 8080 --target-port 80
kubectl -n dojo-dns-service exec deployment/client -- wget -qO- http://api:8080/
```
