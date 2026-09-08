# Worked solution

## LoadBalancer

```bash
kubectl -n dojo-services-types patch svc web -p '{"spec":{"type":"LoadBalancer"}}'
```

`kubectl edit`, or `kubectl expose ... --type=LoadBalancer` on a fresh Service,
do the same.

```bash
kubectl -n dojo-services-types get svc web
```

```
NAME   TYPE           CLUSTER-IP      EXTERNAL-IP   PORT(S)        AGE
web    LoadBalancer   10.106.12.44    <pending>     80:31337/TCP   1m
```

`<pending>` forever. That is not a broken cluster: `type: LoadBalancer` is a
*request*, and something has to satisfy it. On a cloud provider, the cloud
controller manager sees the Service, provisions a real load balancer, and
writes its address into `status.loadBalancer.ingress`. On bare VMs there is no
cloud controller manager and nothing else claiming the job, so the field is
never filled in. Installing something like MetalLB is what makes it resolve.

The important part is what you get anyway. The three types nest:

```text
ClusterIP     ── a virtual IP inside the cluster
NodePort      ── ClusterIP + a port on every node
LoadBalancer  ── NodePort + an external address, if someone provides one
```

So `web` is still reachable from outside, on the node port it was allocated
(`31337` above, in the 30000-32767 range):

```bash
kubectl -n dojo-services-types get svc web -o jsonpath='{.spec.ports[0].nodePort}'
curl http://worker1:31337
```

This is why "the EXTERNAL-IP is pending" is rarely the actual problem in front
of you. The Service works; only the convenience address is missing.

## ExternalName

```bash
kubectl -n dojo-services-types create service externalname api-alias \
  --external-name=kubernetes.default.svc.cluster.local
```

Or as YAML:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: api-alias
  namespace: dojo-services-types
spec:
  type: ExternalName
  externalName: kubernetes.default.svc.cluster.local
```

Note what is *not* there: no `selector`, no `ports`, no `clusterIP`. An
ExternalName Service is not a proxy and does not participate in kube-proxy at
all. CoreDNS simply returns a CNAME:

```bash
kubectl -n dojo-services-types exec deployment/dnsprobe -- nslookup api-alias
```

```
Name:      api-alias.dojo-services-types.svc.cluster.local
Address 1: 10.96.0.1 kubernetes.default.svc.cluster.local
```

Because it is DNS and nothing else, two things follow, and both show up as
real-world bugs:

- **The client resolves the target itself**, so the target must be resolvable
  *from the Pod*. Pointing an ExternalName at a name only the nodes can
  resolve produces a Service that looks fine and never works.
- **There is no port remapping.** The client connects to whatever port it
  asked for, on the aliased name. `ports` in the spec is ignored; it is
  accepted only so that tooling which always writes one does not break.

In production this field usually holds an external DNS name --
`db.eu-west-1.rds.amazonaws.com` -- which is the point of the type: a stable
in-cluster name for something that is not in the cluster, changeable without
touching any workload. Aliasing an in-cluster name, as here, is the same
mechanism and happens to be checkable without depending on the internet.

## Checking before you grade

```bash
kubectl -n dojo-services-types get svc
kubectl -n dojo-services-types get endpointslices
```

`web` as LoadBalancer with a node port and two endpoints; `api-alias` with
`<none>` for its cluster IP and no EndpointSlice of its own.
