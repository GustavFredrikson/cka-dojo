# Give a workload an identity

Work in namespace `dojo-sa-token`.

1. Create a ServiceAccount named `reporter`.
2. Grant it `get`, `list` and `watch` on Pods in this namespace, and nothing
   else.
3. Create Pod `consumer` (`busybox:1.36`, `sleep 3600`) that runs **as**
   `reporter` and receives a **projected** ServiceAccount token:
   - volume named `api-token`
   - audience `dojo-metrics`
   - expiry `3600` seconds
   - mounted so the token lands at `/var/run/secrets/dojo/token`

Then confirm for yourself that a token issued for `reporter` really works:

```
kubectl -n dojo-sa-token create token reporter --duration=10m
```
