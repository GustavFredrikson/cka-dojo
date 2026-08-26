# Extend the API with a namespaced custom resource

Create a namespaced CRD with:

- group `dojo.cka`;
- version `v1`, served and storage;
- kind `Backup`, plural `backups`, singular `backup`;
- structural schema requiring `spec.schedule` as a string.

Then create `Backup` named `nightly` in namespace `dojo-crd-exam` with
`spec.schedule: "0 2 * * *"`.
