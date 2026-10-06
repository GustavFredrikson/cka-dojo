# Worked solution

## Working it out

Ask the node why it is not Ready:

```bash
kubectl describe node cp1 | grep -A5 'Conditions:'
```

`Ready` is `False` with reason `KubeletNotReady` and the message
`container runtime network not ready: NetworkReady=false ... cni plugin not
initialized`. The kubelet looks for a CNI configuration in `/etc/cni/net.d` and
there is nothing there:

```bash
ssh cp1 'ls -la /etc/cni/net.d'
```

CoreDNS being `Pending` is a consequence, not a separate problem. It tolerates
the control-plane taint, so it would schedule — but the scheduler will not place
it on a NotReady node.

This is by design. `kubeadm init` says so in its own output: *"You should now
deploy a pod network to the cluster."* Kubernetes defines the interface and
ships no implementation.

## Fixing it

The CNI is installed like any other workload — a manifest that runs a DaemonSet,
which writes the config to `/etc/cni/net.d` on every node it lands on.

**Flannel** is the least work here, because its default range is already the one
this cluster was built with:

```bash
kubectl apply -f https://github.com/flannel-io/flannel/releases/latest/download/kube-flannel.yml
```

**Calico** works too, but its default pool is `192.168.0.0/16`, so it needs an
edit:

```bash
kubectl create -f https://raw.githubusercontent.com/projectcalico/calico/v3.32.1/manifests/tigera-operator.yaml
curl -O https://raw.githubusercontent.com/projectcalico/calico/v3.32.1/manifests/custom-resources.yaml
# set spec.calicoNetwork.ipPools[0].cidr to 10.244.0.0/16
kubectl create -f custom-resources.yaml
```

Either way:

```bash
kubectl get nodes -w
kubectl -n kube-system get pods
```

`cp1` goes `Ready` within a minute of the DaemonSet Pod starting, and CoreDNS
follows.

## The mistake worth avoiding

Installing a CNI whose range does not match `--pod-network-cidr`. The node goes
Ready, CoreDNS starts, and everything looks finished — but the
controller-manager is handing out `10.244.x` node CIDRs while the CNI allocates
from somewhere else. Pods get addresses the cluster does not route, and it
surfaces later as "some Pods cannot reach each other", which is a far worse
thing to debug than a NotReady node.

Check it directly:

```bash
kubectl -n kube-system get pod -l k8s-app=kube-dns -o wide
kubectl get node cp1 -o jsonpath='{.spec.podCIDR}'; echo
```

The Pod IP must fall inside the node's `podCIDR`. Grading asserts exactly that.

## Checking before you grade

```bash
kubectl get nodes
kubectl -n kube-system get deploy coredns
ssh cp1 'ls /etc/cni/net.d'
kubectl -n kube-system get pod -l k8s-app=kube-dns -o wide
```
