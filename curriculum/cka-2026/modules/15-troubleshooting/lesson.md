# Mixed troubleshooting

## Mental model

Troubleshooting is not guessing components. It is narrowing the failure one
boundary at a time, using evidence from the layer that owns that boundary.

```text
Task symptom
    ↓ reproduce it
Kubernetes object state and events
    ↓ identify the failing layer
Workload → scheduling → service/DNS → node → control plane
    ↓ inspect the owning component
Logs, systemd, configuration and runtime state
    ↓ smallest safe repair
Re-run the original observation
```

Always separate observations from explanations:

- Observation: `worker1` has Ready condition `Unknown`.
- Explanation to test: the kubelet is not reporting.
- Evidence: `systemctl`, `journalctl`, runtime socket and network reachability.

An explanation is not a diagnosis until evidence can distinguish it from the
other causes that produce the same symptom.

## Objects involved

At the Kubernetes layer:

- Pod phase, container states and restart counts.
- Events on Pods, workloads, claims and nodes.
- Node conditions, taints and `spec.unschedulable`.
- Deployments, Services, EndpointSlices, DNS and NetworkPolicy.

At the node layer:

- `kubelet`: registers the node, reports status and drives Pod lifecycle.
- `containerd`: creates containers and exposes the CRI socket.
- CNI: connects Pod network namespaces and programs routes/policy.
- systemd: service lifecycle and whether a repair survives reboot.
- journal: the component's own account of what failed.

## Commands worth knowing

Cluster overview:

```bash
kubectl get nodes -o wide
kubectl get pods -A -o wide
kubectl get events -A --sort-by=.lastTimestamp | tail -30
kubectl describe pod -n NAMESPACE POD
kubectl describe node NODE
```

On a node:

```bash
ssh worker1
systemctl status kubelet containerd
systemctl is-enabled kubelet containerd
journalctl -u kubelet --no-pager -n 100
journalctl -u containerd --no-pager -n 100
crictl ps -a
crictl pods
```

Control-plane static Pods:

```bash
ssh cp1
sudo ls -l /etc/kubernetes/manifests
sudo crictl ps -a
sudo journalctl -u kubelet --no-pager -n 100
```

## Diagnostic workflow

Use the widest cheap observation first, then narrow.

```text
Is the failure one workload or the whole cluster?
  one workload → describe Pod; events; logs; owner object
  one node     → node conditions; ssh; kubelet/runtime
  networking   → DNS; Service; EndpointSlice; direct Pod test
  cluster-wide → API server; static Pods; etcd; certificates
```

For a NotReady node:

```text
kubectl describe node
  ↓ condition reason/message and last heartbeat
ssh node
  ↓ systemctl status kubelet containerd
journalctl the unhealthy component
  ↓ configuration, socket, certificate, disk or network evidence
repair
  ↓ enable service if the fix must survive reboot
kubectl get node until Ready
```

Do not restart every service at once. That destroys evidence and teaches you
nothing about which repair mattered. Inspect first, change one thing, then
verify through the same path that exposed the failure.

## Common CKA failure modes

- Kubelet stopped, disabled, misconfigured or using an expired credential.
- Container runtime stopped or kubelet points at the wrong CRI socket.
- Pod is Pending because of requests, taints, affinity or an unbound claim.
- Pod is in CrashLoopBackOff because the process exits, probes fail, or config
  is absent.
- Service has no endpoints or sends traffic to the wrong port.
- CoreDNS Pods or configuration are unhealthy.
- CNI configuration or node routes prevent Pod traffic.
- A control-plane static Pod manifest contains a bad flag, path or certificate.
- A repair starts a service but does not enable it, so reboot reintroduces the
  fault.

## 5-minute walkthrough

Build a repeatable node-health snapshot without changing anything:

```bash
kubectl get nodes
kubectl get node worker1 \
  -o jsonpath='{range .status.conditions[*]}{.type}{"="}{.status}{" "}{.reason}{"\n"}{end}'
ssh worker1 'systemctl is-active kubelet containerd'
ssh worker1 'systemctl is-enabled kubelet containerd'
ssh worker1 'sudo crictl ps | head'
```

Then inspect recent kubelet evidence:

```bash
ssh worker1 'sudo journalctl -u kubelet --since "10 minutes ago" --no-pager | tail -40'
```

The purpose is not to memorise normal output. It is to know where normal state
lives, so abnormal state is obvious under time pressure.

## Labs

`node-not-ready` produces the same API-level symptom from more than one node
failure. Use node evidence to distinguish them, and make the repair persistent.

```bash
dojo start node-not-ready
```
