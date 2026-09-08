# Solution

## Working it out

The shape of the symptom is the clue: cluster names fail, external names do
not. Prove that before theorising:

```bash
kubectl -n dojo-coredns-corefile exec deployment/dnsprobe -- nslookup web
kubectl -n dojo-coredns-corefile exec deployment/dnsprobe -- nslookup kubernetes.io
```

The second one works. So the client's `resolv.conf` is fine, the Service IP
routes, CoreDNS is up, and it is forwarding upstream queries correctly. The
only thing failing is CoreDNS answering *authoritatively* for the cluster --
which is one plugin.

CoreDNS keeps its entire configuration in one ConfigMap:

```bash
kubectl -n kube-system get configmap coredns -o jsonpath='{.data.Corefile}'
```

```
.:53 {
    errors
    health {
       lameduck 5s
    }
    ready
    kubernetes cluster.internal in-addr.arpa ip6.arpa {
       pods insecure
       fallthrough in-addr.arpa ip6.arpa
       ttl 30
    }
    prometheus :9153
    forward . /etc/resolv.conf
    cache 30
    loop
    reload
    loadbalance
}
```

`kubernetes cluster.internal`. The first argument to the `kubernetes` plugin
is the cluster domain it is authoritative for. Now look at what the clients
ask for:

```bash
kubectl -n dojo-coredns-corefile exec deployment/dnsprobe -- cat /etc/resolv.conf
```

```
search dojo-coredns-corefile.svc.cluster.local svc.cluster.local cluster.local
```

Every Pod asks for `...cluster.local`, because the kubelet's
`--cluster-domain` says so. CoreDNS is authoritative for `cluster.internal`
instead, so those queries fall through to `forward . /etc/resolv.conf`, out to
the upstream resolver, and come back NXDOMAIN. Nothing errors; the query is
answered, just not by the plugin that knows the answer.

This is worth internalising as a pattern: **two halves of DNS configured
independently, and only one was changed.** The other direction happens too --
a kubelet reconfigured with a different `--cluster-domain` against an
untouched Corefile produces exactly the same failure.

## Fixing it

```bash
kubectl -n kube-system edit configmap coredns
```

Change `cluster.internal` back to `cluster.local` on the `kubernetes` line.
Non-interactively:

```bash
kubectl -n kube-system get configmap coredns -o yaml \
  | sed 's/cluster\.internal/cluster.local/g' \
  | kubectl apply -f -
```

Then make it take effect. The `reload` plugin watches the file and picks up
changes on its own, but the ConfigMap has to reach the Pod's filesystem first
(kubelet's sync period) and then CoreDNS has to notice -- a minute or two in
total. If you would rather not wait:

```bash
kubectl -n kube-system rollout restart deployment coredns
kubectl -n kube-system rollout status deployment coredns
```

Restarting is not a workaround here, it is just faster. But knowing that
`reload` exists is what stops you concluding your edit did not work when it
simply has not landed yet.

## Why the reverse-lookup check is there

`in-addr.arpa` and `ip6.arpa` are the plugin's other two zones, on the same
line. A fix that replaces the line rather than editing the one word -- or that
drops the `fallthrough` directive below it -- restores forward resolution and
quietly breaks reverse lookups for Service and Pod addresses. Grading checks
both directions so that shortcut does not pass.

## Checking before you grade

```bash
kubectl -n kube-system get configmap coredns -o jsonpath='{.data.Corefile}' | head -12
kubectl -n kube-system get pods -l k8s-app=kube-dns
kubectl -n dojo-coredns-corefile exec deployment/dnsprobe -- nslookup web
kubectl -n dojo-coredns-corefile exec deployment/dnsprobe -- \
  nslookup "$(kubectl -n dojo-coredns-corefile get svc web -o jsonpath='{.spec.clusterIP}')"
```

If the forward lookup works and the reverse one does not, compare your
`kubernetes` block against the original above -- the zones and the
`fallthrough` line are load-bearing.
