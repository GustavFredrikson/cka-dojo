# Workloads

## Mental model

```text
Deployment
    ↓ owns
ReplicaSet
    ↓ owns
Pods
    ↓ consume
ConfigMaps + Secrets
```

A Deployment declares the desired Pod template and replica count. Controllers
continually reconcile actual state toward that declaration. Rollouts create a
new ReplicaSet; rollback selects an earlier template revision.

## Objects involved

- `Deployment`: rollout strategy, replicas and Pod template.
- `ReplicaSet`: one revision's Pod population.
- `Pod`: the scheduled containers and their current state.
- `ConfigMap` and `Secret`: configuration referenced as environment or files.
- Events and logs: evidence explaining why declared state is not healthy.

## Commands worth knowing

```bash
kubectl create deployment web --image=nginx:1.27-alpine --replicas=3
kubectl scale deployment web --replicas=4
kubectl set image deployment/web web=nginx:1.27-alpine
kubectl rollout status deployment/web
kubectl rollout history deployment/web
kubectl rollout undo deployment/web
kubectl logs pod-name
kubectl logs pod-name --previous
kubectl describe pod pod-name
kubectl get events --sort-by=.metadata.creationTimestamp
```

Use `kubectl create ... --dry-run=client -o yaml` to generate small manifests
instead of recalling every field.

## Diagnostic workflow

```text
Deployment unavailable
  ↓ get deployment, replicasets and pods
No Pod created?     → selector, quota, admission
Pod Pending?        → describe; scheduling or volume evidence
ImagePullBackOff?   → image name and registry access
Config error?       → referenced ConfigMap or Secret
CrashLoopBackOff?   → logs and logs --previous
Running, not Ready? → readiness probe and application endpoint
```

## Common CKA failure modes

- The container image or command is wrong.
- A required ConfigMap or Secret is missing or named incorrectly.
- Deployment selectors do not match Pod-template labels.
- A rollout changes the wrong container name.
- Resource requests make Pods unschedulable.
- A liveness or readiness probe targets the wrong path or port.
- The learner inspects only the Deployment and misses Pod events/logs.

## 5-minute walkthrough

```bash
kubectl create namespace demo-workloads
kubectl -n demo-workloads create deployment web --image=nginx:1.26-alpine --replicas=2
kubectl -n demo-workloads rollout status deployment/web
kubectl -n demo-workloads set image deployment/web nginx=nginx:1.27-alpine
kubectl -n demo-workloads rollout history deployment/web
kubectl -n demo-workloads rollout undo deployment/web
kubectl delete namespace demo-workloads
```

## Labs

```bash
dojo start workloads-follow
dojo start workloads-build
dojo start workloads-config
dojo start workloads-inspect
dojo start workloads-rollout
dojo start workloads-guided-crashloop
dojo start workloads-broken-app
```
