# Solution

## Working it out

The instinct is to look at the DaemonSet's spec. Look at its metadata first:

```bash
kubectl -n calico-system get ds calico-node -o jsonpath='{.metadata.ownerReferences}' | jq
```

```json
[{"apiVersion":"operator.tigera.io/v1","kind":"Installation","name":"default",
  "controller":true,"blockOwnerDeletion":true}]
```

`controller: true` is the important word. This DaemonSet is not something a
person wrote; it is *output*. An `Installation` custom resource declares what
the CNI should look like, and a controller — the Tigera operator, in
`tigera-operator` — continuously makes the cluster match it. Your edit did not
fail and was not rejected: it was noticed and undone, which is the operator
doing exactly its job.

```bash
kubectl -n tigera-operator get deploy
kubectl get installations.operator.tigera.io
kubectl get installation default -o yaml | head -40
```

This is the operator pattern, and it is worth naming the parts because they
recur in every operator you will meet:

- a **CRD** extends the API with a new kind (`Installation`);
- a **custom resource** is the desired state, written by you;
- a **controller** watches it and reconciles reality towards it;
- everything the controller creates carries an `ownerReference` back to the
  CR — which is both how it finds its own objects and how garbage collection
  cleans them up if the CR is deleted.

The operational rule that follows: **never edit what an operator owns.** The
`ownerReferences` field is how you tell, and it takes one command.

Helm is the useful contrast. Helm renders templates and applies them once; if
you edit the result, your edit stays until the next `helm upgrade`. An
operator is a running process, so your edit lasts until it next looks —
seconds. Both install things; only one keeps watching.

## Fixing it

Find the field on the CR rather than guessing:

```bash
kubectl explain installation.spec.nodeUpdateStrategy
kubectl explain installation.spec.nodeUpdateStrategy.rollingUpdate
```

Then declare it:

```bash
kubectl patch installation default --type=merge \
  -p '{"spec":{"nodeUpdateStrategy":{"rollingUpdate":{"maxUnavailable":2}}}}'
```

`kubectl edit installation default` does the same thing.

Watch it propagate — this is the reconciliation loop running in your favour
for once:

```bash
kubectl -n calico-system get ds calico-node \
  -o jsonpath='{.spec.updateStrategy.rollingUpdate.maxUnavailable}' ; echo
```

It becomes `2` within a few seconds. Nothing restarts:
`nodeUpdateStrategy` only describes how a *future* rollout should proceed, so
no calico-node Pod is replaced and pod networking is untouched. That is worth
checking rather than assuming, because plenty of fields on this CR do trigger
a CNI rollout.

## Checking before you grade

```bash
kubectl get installation default -o jsonpath='{.spec.nodeUpdateStrategy}' ; echo
kubectl -n calico-system get ds calico-node \
  -o jsonpath='{.spec.updateStrategy.rollingUpdate.maxUnavailable}' ; echo
kubectl -n calico-system rollout status daemonset/calico-node
kubectl get nodes
```

Both objects should read `2`. Grading checks both on purpose: the CR alone
would prove you wrote a declaration, and the DaemonSet agreeing is what proves
the operator accepted it and did the work — which is the only way that field
can hold the value at all.
