# Control plane and maintenance

## Mental model

On a kubeadm control plane, kubelet watches `/etc/kubernetes/manifests` and
runs API server, scheduler, controller-manager and etcd as static Pods. Their
mirror Pods appear through the API, but the files on the node are authoritative.

## Objects involved

- Static Pod manifests and mirror Pods.
- kube-scheduler and pending unscheduled Pods.
- Node cordon state, eviction and PodDisruptionBudgets.
- kubelet and container runtime services.

## Commands worth knowing

```bash
ssh cp1
sudo ls -l /etc/kubernetes/manifests
sudo crictl ps -a
kubectl get pods -n kube-system -o wide
kubectl cordon worker1
kubectl drain worker1 --ignore-daemonsets --delete-emptydir-data
kubectl uncordon worker1
```

## Diagnostic workflow

Separate API failure from scheduling failure. If the API works but new Pods
stay Pending without a scheduling decision, inspect scheduler health and its
static manifest. During maintenance, verify both node state and workload
placement.

## Common CKA failure modes

- A static manifest is missing or malformed.
- A component flag references a wrong file or address.
- A worker is drained but never uncordoned.
- DaemonSets or local storage make a drain require explicit flags.
- A repair changes the mirror Pod rather than the source manifest.

## 5-minute walkthrough

```bash
kubectl get pods -n kube-system -o wide
ssh cp1
sudo ls /etc/kubernetes/manifests
sudo crictl ps --name kube-scheduler
```

## Labs

```text
dojo start controlplane-static-pod
dojo start controlplane-scheduler
dojo start node-drain
```
