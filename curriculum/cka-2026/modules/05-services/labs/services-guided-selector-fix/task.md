# Repair a known selector mismatch

In namespace `dojo-services-guided`, Deployment `web` is healthy but Service
`web` has no endpoints.

The Service selector is wrong. Compare it with the Pod labels and correct the
existing Service so traffic works again.
