# Worked solution

```bash
helm template shop /home/student/cka-assets/helm --set replicaCount=3 --set service.port=8080
helm install shop /home/student/cka-assets/helm -n dojo-helm-exam \
  --set replicaCount=3 --set service.port=8080
helm status shop -n dojo-helm-exam
```
