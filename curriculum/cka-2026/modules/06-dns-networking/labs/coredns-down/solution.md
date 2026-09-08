# Solution

## Working it out

The first useful fact is in the task: *every* namespace is affected. That rules
out anything you own and points at the one piece of infrastructure every Pod
depends on. Confirm what that is:

```bash
kubectl -n dojo-coredns-down exec deployment/dnsprobe -- cat /etc/resolv.conf
```

```
nameserver 10.96.0.10
search dojo-coredns-down.svc.cluster.local svc.cluster.local cluster.local
```

`10.96.0.10` is a Service IP -- the tenth address of the service CIDR, which
is where kubeadm always puts cluster DNS. So the chain to check is the same one
you would check for any Service that does not answer, and in the same order:

```bash
kubectl -n kube-system get deployment coredns
kubectl -n kube-system get svc kube-dns
kubectl -n kube-system get endpointslices -l kubernetes.io/service-name=kube-dns
kubectl -n kube-system get pods -l k8s-app=kube-dns
```

Two different things produce the identical symptom, and this listing tells them
apart immediately.

**No Pods at all.** `kubectl get deployment coredns` reads `0/0`. Nobody is
serving; the Service has no endpoints because there is nothing to select.

**Pods running, no endpoints.** The Deployment reads `2/2` and the Pods are
`Running`, but the EndpointSlice is empty. Then the Service is selecting
something the Pods do not carry:

```bash
kubectl -n kube-system get svc kube-dns -o jsonpath='{.spec.selector}'
kubectl -n kube-system get pods -l k8s-app=kube-dns --show-labels
```

`{"k8s-app":"kube-dns-retired"}` against Pods labelled `k8s-app=kube-dns`.

Note the naming, because it catches people out: the Deployment is called
`coredns` and the Service in front of it is called `kube-dns`. The Service kept
the old name so that nothing pointing at cluster DNS had to change when the
implementation did.

## Fixing it

For no replicas:

```bash
kubectl -n kube-system scale deployment coredns --replicas=2
kubectl -n kube-system rollout status deployment coredns
```

Two, not one -- kubeadm's default, and DNS is the last thing you want on a
single Pod.

For the selector:

```bash
kubectl -n kube-system patch service kube-dns \
  -p '{"spec":{"selector":{"k8s-app":"kube-dns"}}}'
```

`kubectl -n kube-system edit svc kube-dns` does the same. What you must not do
is relabel the Pods to match the Service: the Deployment's own
`selector.matchLabels` is immutable and still says `k8s-app=kube-dns`, so you
would be fighting the ReplicaSet. Fix the side that is wrong.

## Checking before you grade

```bash
kubectl -n kube-system get svc kube-dns -o wide
kubectl -n kube-system get endpointslices -l kubernetes.io/service-name=kube-dns
kubectl -n dojo-coredns-down exec deployment/dnsprobe -- \
  nslookup web.dojo-coredns-down.svc.cluster.local
```

Two addresses in the EndpointSlice, and an answer from `nslookup`. Resolution
recovers within a few seconds of the endpoints appearing -- there is nothing to
restart on the client side, because `resolv.conf` never changed.
