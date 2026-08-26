# Worked solution

Set `namePrefix: prod-`, replica count `3`, and the label pair
`environment: prod`, then:

```bash
kubectl kustomize /home/student/cka-assets/kustomize/overlays/prod
kubectl apply -k /home/student/cka-assets/kustomize/overlays/prod
```
