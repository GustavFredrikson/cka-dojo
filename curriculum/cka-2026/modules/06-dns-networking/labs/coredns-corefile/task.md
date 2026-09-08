# Service names stop resolving, external ones still work

In namespace `dojo-coredns-corefile`, Pod `dnsprobe` cannot resolve the `web`
Service, by short name or by fully-qualified name. The same is true in every
namespace, including `kube-system`.

CoreDNS itself is healthy: two Pods, both Ready, no restarts, and it still
answers for names outside the cluster.

Its configuration was edited. Find what was changed and put cluster DNS back
to serving this cluster's own domain, so that:

- `nslookup web` works from a Pod in that namespace, and
- a reverse lookup of the `web` Service's cluster IP still works.
