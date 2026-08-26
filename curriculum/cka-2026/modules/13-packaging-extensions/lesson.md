# Packaging and extensions

## Mental model

Kustomize transforms YAML without templates. Helm renders a chart and records
a release. CRDs extend the Kubernetes API; custom resources are then validated
and stored like built-in objects.

## Objects involved

- Kustomization bases, overlays, patches and generated output.
- Helm charts, values, releases and revisions.
- CustomResourceDefinition schemas and custom resources.

## Commands worth knowing

```bash
kubectl kustomize DIR
kubectl apply -k DIR
helm install RELEASE CHART -n NAMESPACE
helm upgrade RELEASE CHART -n NAMESPACE --set key=value
helm list -A
kubectl get crd
kubectl api-resources
```

## Diagnostic workflow

Render before applying. For Kustomize, inspect `kubectl kustomize`; for Helm,
inspect `helm template`, values and release status. For CRDs, wait for the API
to establish the definition before creating a custom resource.

## Common CKA failure modes

- Editing generated output instead of the base or overlay.
- Installing a Helm release into the wrong namespace.
- Supplying the wrong value path to a chart.
- Using a CR kind, group or version that does not match the CRD.
- Omitting a structural OpenAPI schema.

## 5-minute walkthrough

```bash
kubectl kustomize ./overlay
helm template demo ./chart
kubectl get crd
kubectl api-resources
```

## Labs

```text
dojo start kustomize-release
dojo start helm-release
dojo start crd-resource
```
