# DNS and Pod networking

## Mental model

Pods use the cluster DNS Service. A Service named `api` in namespace `backend`
has the stable name `api.backend.svc.cluster.local`; search domains allow the
short name `api` only from the same namespace.

## Objects involved

- Services and EndpointSlices provide stable discovery targets.
- CoreDNS answers cluster-zone queries.
- Pod `dnsPolicy` decides whether cluster DNS is used.
- `/etc/resolv.conf` shows the nameserver and search path a Pod received.

## Commands worth knowing

```bash
kubectl get svc,endpointslice -A
kubectl exec POD -- cat /etc/resolv.conf
kubectl exec POD -- nslookup kubernetes.default
kubectl -n kube-system get deploy,svc,pods -l k8s-app=kube-dns
kubectl -n kube-system logs deployment/coredns
```

## Diagnostic workflow

Test the Service by ClusterIP, then its same-namespace name, then its full
cross-namespace name. Inspect EndpointSlices before blaming DNS. If every name
fails, inspect Pod DNS policy and CoreDNS health.

## Common CKA failure modes

- A short Service name is used from another namespace.
- The Service selector produces no endpoints.
- `dnsPolicy: Default` bypasses cluster DNS.
- CoreDNS is unavailable or has invalid configuration.
- NetworkPolicy blocks DNS traffic.

## 5-minute walkthrough

```bash
kubectl run dns --image=busybox:1.36 --restart=Never -- sleep 3600
kubectl exec dns -- nslookup kubernetes.default
kubectl exec dns -- cat /etc/resolv.conf
kubectl delete pod dns
```

## Labs

```text
dojo start dns-service-discovery
dojo start dns-cross-namespace
dojo start dns-policy
```
