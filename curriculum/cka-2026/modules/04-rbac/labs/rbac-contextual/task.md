# A workload identity cannot inspect Deployments

ServiceAccount `deploy-reader` in namespace `dojo-rbac-contextual` should have
only `get`, `list` and `watch` access to Deployments in that namespace.
Authorization currently returns `no`.

Restore the narrow grant. Do not grant delete access or access to Secrets.
