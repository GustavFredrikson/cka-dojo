# Packaging and extensions

## Mental model

Kustomize transforms YAML without templates. Helm renders a chart and records
a release. CRDs extend the Kubernetes API; custom resources are then validated
and stored like built-in objects.

An operator is the fourth thing in that list and behaves unlike the others:
Helm and Kustomize *render and apply* once, so an edit you make afterwards
survives until the next `helm upgrade`. An operator is a running controller
that keeps reconciling, so an edit to anything it owns survives seconds.

```text
CRD            defines a new kind        (Installation)
custom resource   desired state, yours to write
controller     watches it, creates and corrects the real objects
ownerReferences  every object it made points back at the CR
```

The operational rule: **never edit what an operator owns.** `ownerReferences`
with `controller: true` is how you tell, and it costs one command.

## Objects involved

- Kustomization bases, overlays, patches and generated output.
- Helm charts, values, releases and revisions.
- CustomResourceDefinition schemas and custom resources.

## Commands worth knowing

```bash
kubectl get ds NAME -n NS -o jsonpath='{.metadata.ownerReferences}'
kubectl api-resources --api-group=operator.tigera.io
kubectl explain installation.spec
```


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

- Editing a Deployment, DaemonSet or ConfigMap that an operator owns, and
  reading the revert as a cluster fault rather than as the controller working.
- Looking for the operator's configuration in a ConfigMap. It is a custom
  resource, and `kubectl api-resources` finds its group.
- Deleting a CR to "clean up" and taking every object it owns with it —
  `ownerReferences` drive garbage collection.


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
dojo start operator-reconcile
```
