# The other two Service types

Work in namespace `dojo-services-types`. Deployment `web` is running behind a
ClusterIP Service of the same name.

**1.** Change the `web` Service to type `LoadBalancer`, keeping port `80`.
Then answer the checkpoints about what happens on a cluster with no cloud
provider.

**2.** Create a second Service named `api-alias` that is a pure DNS alias --
no cluster IP, no selector, no endpoints -- resolving to:

```
kubernetes.default.svc.cluster.local
```

A Pod in the namespace must be able to resolve `api-alias` and be sent on to
that name.

Answer each checkpoint with `dojo check <answer>`, and `dojo check` on its own
for the configuration steps.
