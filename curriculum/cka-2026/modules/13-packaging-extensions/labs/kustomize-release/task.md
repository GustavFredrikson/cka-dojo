# Correct and apply a provided Kustomize overlay

Files are provided at `/home/student/cka-assets/kustomize` in the terminal VM.
Modify only `overlays/prod/kustomization.yaml` so the rendered Deployment:

- is named `prod-web`;
- has 3 replicas;
- has label `environment=prod` on the Pod template;
- is deployed in namespace `dojo-kustomize-exam`.

Render the overlay to inspect it, then apply it.
