# Services

## Mental model

A Service is a stable virtual address in front of a changing set of Pods. It
does not run the application and it does not continuously probe arbitrary Pod
IPs. It selects Pods by label, the control plane writes those backends into
EndpointSlices, and the cluster's networking sends Service traffic to one of
those endpoints.

```text
Client
  ↓ DNS name or ClusterIP
Service
  ↓ selector matches Pod labels
EndpointSlice
  ↓ ready Pod IP + target port
Pod
```

That gives you two separate questions:

1. Does the Service discover the correct Pods?
2. Does traffic reach the port where the application listens?

A Service object can look perfectly healthy while both answers are “no.”

## Objects involved

- `Service`: stable address, selector, exposed `port`, destination
  `targetPort`.
- `Pod`: labels, container ports and readiness.
- `EndpointSlice`: the actual ready addresses and ports selected for a
  Service.
- CoreDNS: resolves names such as `web.shop.svc.cluster.local` to a Service
  address.

`port` is the port clients use on the Service. `targetPort` is where the
Service sends traffic on a Pod. A named `targetPort` must match a named
container port exactly.

## Commands worth knowing

```bash
kubectl get svc -A
kubectl -n shop describe svc web
kubectl -n shop get pods --show-labels
kubectl -n shop get endpointslice \
  -l kubernetes.io/service-name=web -o wide
kubectl -n shop get svc web -o yaml
kubectl -n shop get deploy web \
  -o jsonpath='{.spec.template.spec.containers[0].ports}'
```

Test from inside the cluster, because a ClusterIP is normally not reachable
from the workstation:

```bash
kubectl -n shop run probe --rm -it \
  --image=busybox:1.36 --restart=Never -- \
  wget -qO- http://web:80
```

## Diagnostic workflow

Start at the Service and move one hop at a time.

```text
Can the client resolve the name?
  no  → inspect DNS and namespace
  yes ↓
Does the Service have ready EndpointSlice addresses?
  no  → compare selector with Pod labels and readiness
  yes ↓
Do EndpointSlice ports point at the application's listening port?
  no  → compare targetPort with the container port
  yes ↓
Does a direct Pod request work?
  no  → application, readiness or Pod networking
  yes ↓
Service path, NetworkPolicy or kube-proxy/dataplane
```

Do not begin by deleting and recreating the Service. First identify which hop
is broken; otherwise a lucky repair teaches you nothing and may discard fields
the task required you to preserve.

## Common CKA failure modes

- Service selector differs from Pod labels by one key or value.
- `targetPort` points at the wrong number.
- A named `targetPort` does not exist on the selected Pods.
- Pods match but are not Ready, so endpoints are not ready for traffic.
- The client uses the short DNS name from another namespace.
- A NetworkPolicy allows Pods but not the actual source or port.
- The task requires preserving the Service type or exposed port, but the
  repair recreates it with defaults.

## 5-minute walkthrough

Create a tiny application and expose it:

```bash
kubectl create namespace demo
kubectl -n demo create deployment web --image=nginx:1.27-alpine --replicas=2
kubectl -n demo expose deployment web --port=80 --target-port=80
kubectl -n demo get svc,pods,endpointslice
```

See the relationship directly:

```bash
kubectl -n demo get svc web -o jsonpath='{.spec.selector}'; echo
kubectl -n demo get pods --show-labels
kubectl -n demo get endpointslice \
  -l kubernetes.io/service-name=web -o yaml
```

Remove the walkthrough when finished:

```bash
kubectl delete namespace demo
```

## Labs

The Services path removes scaffolding one step at a time:

```bash
dojo start services-follow
dojo start services-build
dojo start services-inspect
dojo start services-guided-selector-fix
dojo start services-no-endpoints
```

The first exercises show a healthy selector and EndpointSlice relationship.
Only the final current exercise hides the cause, with several reproducible
variants so diagnosis—not recall—determines the repair.
