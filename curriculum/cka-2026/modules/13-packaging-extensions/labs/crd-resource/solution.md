# Worked solution

```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: backups.dojo.cka
spec:
  group: dojo.cka
  scope: Namespaced
  names:
    plural: backups
    singular: backup
    kind: Backup
  versions:
    - name: v1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              required: [schedule]
              properties:
                schedule:
                  type: string
---
apiVersion: dojo.cka/v1
kind: Backup
metadata:
  name: nightly
  namespace: dojo-crd-exam
spec:
  schedule: "0 2 * * *"
```

Save the first document as `crd.yaml` and the second as `backup.yaml`. Apply the
CRD first and wait for discovery before applying the custom resource:

```bash
kubectl apply -f crd.yaml
kubectl wait --for=condition=Established crd/backups.dojo.cka
kubectl apply -f backup.yaml
```
