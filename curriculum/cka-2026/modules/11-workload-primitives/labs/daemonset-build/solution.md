# Worked solution

## Working it out

There is no `--dry-run` generator for a DaemonSet. The quickest honest route
is to generate a Deployment and change two things:

```bash
kubectl -n dojo-daemonset-build create deployment node-agent \
  --image=busybox:1.36 --dry-run=client -o yaml -- sleep 3600 > ds.yaml
```

Then edit: `kind: DaemonSet`, and delete `spec.replicas` and
`spec.strategy` -- a DaemonSet has neither. `kubectl explain daemonset.spec`
confirms what is left.

## The part the task is actually about

A DaemonSet with no tolerations reports:

```
NAME         DESIRED   CURRENT   READY
node-agent   2         2         2
```

Two, not three, and nothing anywhere says "error". The control-plane node
carries a taint:

```bash
kubectl describe node cp1 | grep -i taints
Taints: node-role.kubernetes.io/control-plane:NoSchedule
```

`desiredNumberScheduled` excludes nodes the Pod could not schedule onto, so
the DaemonSet considers itself complete. This is the failure mode to
recognise: for a DaemonSet, "everything is Ready" and "everything is running"
are different claims.

Tolerate it:

```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: node-agent
  namespace: dojo-daemonset-build
spec:
  selector:
    matchLabels: {app: node-agent}
  template:
    metadata:
      labels: {app: node-agent}
    spec:
      tolerations:
        - key: node-role.kubernetes.io/control-plane
          operator: Exists
          effect: NoSchedule
      containers:
        - name: agent
          image: busybox:1.36
          command: ["sleep", "3600"]
```

`operator: Exists` with no `value` tolerates the taint whatever its value,
which is what you want for a key whose value you do not control. The blunter
version -- `operator: Exists` with no `key` either -- tolerates *every* taint,
including `NoExecute` ones a node is using to shed work. Do not reach for it
by habit.

## Checking before you grade

```bash
kubectl -n dojo-daemonset-build get daemonset node-agent
kubectl -n dojo-daemonset-build get pods -o wide
```

Three Pods, one per node, and `cp1` among them.
