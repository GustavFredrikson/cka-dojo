# Explain what an Ingress will and will not match

Ingress `storefront` in namespace `dojo-ingress-inspect` is already routing
traffic for two Services. Read it rather than changing it.

Use `kubectl describe ingress storefront -n dojo-ingress-inspect` and the
`client` Pod to try requests, then answer each checkpoint with `dojo check`.

The two rules do **not** use the same `pathType`, and that difference is the
point of the exercise.
