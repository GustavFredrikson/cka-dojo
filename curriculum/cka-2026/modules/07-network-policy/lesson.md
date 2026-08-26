# NetworkPolicy

## Mental model

NetworkPolicy is allow-list logic for selected Pods. Once a Pod is isolated
for ingress, traffic is denied unless at least one applicable policy allows it.

```text
source Pod labels/namespace
          ↓ ingress.from
NetworkPolicy selects destination Pods
          ↓ ports
destination Pod
```

Policies are additive: matching one allow rule is enough. A policy that selects
no destination Pods protects nothing.

## Objects involved

- Pod labels: identify both destination and allowed source.
- Namespace labels: permit sources by namespace.
- `NetworkPolicy`: namespaced selectors, directions and allowed peers/ports.
- CNI: enforces policy; this environment uses Calico.

## Commands worth knowing

```bash
kubectl get networkpolicy -A
kubectl describe networkpolicy allow-client -n app
kubectl get pods --show-labels -n app
kubectl exec deploy/client -n app -- wget -T 2 -qO- http://server
kubectl get namespace --show-labels
```

Always test from a known source Pod. Testing from the terminal VM does not
exercise the same policy identity.

## Diagnostic workflow

```text
Pod-to-Service request fails
  ↓ verify Service endpoints and direct application health
Which policy selects the destination Pod?
  none → policy is not the cause
  yes  ↓
Does ingress.from match source Pod and namespace labels?
  no → repair peer selectors
  yes ↓
Does the allowed protocol/port match traffic?
  no → repair port rule
  yes ↓
Inspect egress policy on the source and DNS allowance
```

## Common CKA failure modes

- `podSelector` selects the wrong destination.
- Source labels do not match `ingress.from.podSelector`.
- `namespaceSelector` and `podSelector` are placed as separate peers (OR)
  instead of one combined peer (AND).
- Port 8080 is allowed while the Service reaches port 80.
- Default deny is created without the intended allow policy.
- Egress isolation blocks DNS and makes a network problem look like DNS failure.

## 5-minute walkthrough

Create two labelled Pods and a Service, test traffic, apply default-deny
ingress, retest, then add one allow policy. Use `kubectl describe` and repeated
source-specific requests to connect selectors with observed behavior.

## Labs

```bash
dojo start netpol-follow
dojo start netpol-build
dojo start netpol-inspect
dojo start netpol-guided
dojo start netpol-contextual
```
