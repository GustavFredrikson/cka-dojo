# Authoring labs

A lab is a directory. No Go, no shell, no per-lab grader code.

```
labs/services-no-endpoints/
├── lab.yaml          # what to build, what to break, what counts as fixed
├── task.md           # what the learner sees
├── solution.md       # never leaves the host
└── manifests/
    └── baseline.yaml
```

Run `dojo content validate` after every change. It catches unknown fault and
grader types, missing manifests, undeclared skills, labs with no hints, and
YAML that will not parse — offline, in under a second.

## The four rules

**1. Every skill climbs concept → command → diagnosis.**

Do not introduce a concept through blind troubleshooting. Begin with a healthy
system and explicit commands, remove scaffolding over several exercises, then
hide the cause. `learningStage` records where an exercise sits:

```text
follow → build → inspect → guided-fix → contextual-fix → diagnose → exam
```

Learning stage is independent of `difficulty`: a detailed walkthrough of
kubelet internals may be advanced but still be a Follow exercise.

**2. Reveal only what is appropriate for the stage.**

At `guided-fix`, naming the broken selector is teaching. At `diagnose`, it
would give away the exercise. A contextual or diagnostic task states the
symptom, never the cause:

> The Pods are healthy. Requests sent to the Service do not reach them.

not

> The Service selector does not match the Pod labels.

Diagnosis *is* the exercise. If the task names the broken field, all that is
left is typing.

**3. Grade state, never commands.**

Never ask "did they run `kubectl patch`". Ask "does the Service now have ready
endpoints". Imperative kubectl, `kubectl edit`, a rewritten manifest and a
patch are all correct, and the exam scores them all the same.

**4. Every fault must be undoable.**

`dojo reset` calls each fault's repair. Prefer a named primitive, which knows
how to undo itself. `nodeExec` requires you to supply the `undo` script, and
validation rejects it if you do not.

Reset is `Teardown` + `Setup`: it repairs faults and deletes the objects in
`setup.apply`, then applies them again. Two consequences bite in practice, and
neither is caught by `content validate` — only by running the loop at the
bottom of this page.

*Cluster-scoped objects the learner creates survive reset.* Deleting the lab's
namespace sweeps up everything inside it, but a PersistentVolume, a
StorageClass or a ClusterRole is not in any namespace and is not in your
manifests either. The lab then still passes immediately after a reset, which
silently hands out a free pass. Ship such objects in the baseline in a state
that fails grading, so the engine owns them and reset restores the failure.

*`waitReady` runs after every manifest is applied, not between them.* If two
workloads in one baseline compete for the same nodes, whichever the scheduler
reaches first wins and the symptom can invert from run to run. When a lab
depends on one workload settling before another appears, put the second one in
a `kubernetesApply` fault: faults land after `waitReady`.

*`waitReady` is `kubectl rollout status`.* So it works for Deployments,
StatefulSets and DaemonSets, and there is nothing for it to wait on for a bare
Pod, a Job or a Service — `rollout status pod/x` is an error, not a wait. A lab
whose baseline is a single Pod simply omits `waitReady`.

*A named Kubernetes fault cannot put back an object the lab does not own.*
`kubernetesPatch`, `kubernetesDelete` and `kubernetesScale` all have a no-op
`Repair` (`internal/fault/kubernetes.go`): they are undoable only because
reset re-applies `setup.apply` afterwards. Point one at something the lab did
not ship — the local-path provisioner, a kube-system Deployment, anything
shared — and the change is permanent, for every later lab on that profile too.
Break shared components with a `nodeExec` that stashes the original state
conditionally and restores it in `undo`. Nothing in `content validate` catches
this.

## Three things about fault scripts

*`nodeExec` scripts get `set -euo pipefail` whether you write it or not* — and
so does `undo`. The engine prepends it to both (`internal/fault/node.go`).
Writing `set -euo pipefail` at the top of a `script:` is therefore harmless and
still worth doing, because it says out loud what the script is running under.

What this means for `undo` is less obvious and matters more: teardown runs
under `errexit` too, so the first command that fails abandons the rest of the
cleanup. Writing `set -uo pipefail` on line 1 of an `undo:` *does* turn
`errexit` back off — but do not rely on reading it that way. Guard the commands
that are allowed to fail explicitly:

```yaml
undo: |
  systemctl start kubelet || true
  rm -f /var/lib/dojo/marker
```

*`command` graders get no `set -e` at all.* The grader builds only
`export KUBECONFIG=…` followed by your command (`internal/grader/node.go`), so
in a multi-line `command:` every exit code but the last is discarded. Start
multi-line grading commands with `set -euo pipefail`, and give every `command`
check a `stdout:` match — the exec error is discarded too, and a zero-value
result has exit code 0, so a check with neither `stdout:` nor `exitCode:`
passes green when it never ran.

Stdin is closed for all of these, which is deliberate (a script read from stdin
would have its tail swallowed by the first command that reads stdin). A heredoc
*inside* the script is unaffected, so `kubectl apply -f - <<'EOF'` works
normally.

*An `undo` that restores a backup must not be able to enshrine a broken
state.* `reset` is teardown followed by setup, so a `script:` that
unconditionally copies the current file over its own backup will capture
whatever the last attempt left behind. Either make the copy conditional on the
current state being good, or reconstruct from a source of truth the learner
cannot damage. `kubeconfig-repair` does the first — and puts the backup in a
lab-level fault, so it is injected before whichever variant runs and repaired
after it.

## lab.yaml

```yaml
schemaVersion: 1
id: services-no-endpoints          # unique across the whole curriculum
title: A Service that does not answer
learningStage: contextual-fix       # progression, separate from difficulty
prerequisites: [services-guided-selector-fix]
# masteryPrerequisites: [services-target-port]
domain: [services-networking, troubleshooting]
skills: [services, selectors, endpointslices, service-debugging]
difficulty: 1                      # 1-5
targetMinutes: 6

environment:
  profile: standard

setup:
  apply:
    - manifests/baseline.yaml
  waitReady:                       # settle the baseline before breaking it
    - namespace: shop
      kind: deployment
      name: web
  faults:
    - type: kubernetesPatch
      resource: service/web
      namespace: shop
      patch:
        spec:
          selector:
            app: web-frontend

grading:
  all:                             # every check must pass
    - type: deploymentAvailable
      namespace: shop
      name: web
      minReplicas: 2
  any: []                          # optional: at least one must pass

hints:                             # escalating, four is a good number
  - Start from how a Service discovers the Pods it should send traffic to.
  - Look at the EndpointSlices for the Service, not just the Service itself.

exam:
  eligible: true
conflicts:                         # resources this lab monopolises
  - namespace:shop
```

`domain` and `skills` values must exist in `curriculum.yaml`. This is enforced,
because progress is reported per skill and a typo would silently create a new
one.

## Interactive checkpoints

Follow, Build, and Inspect exercises can split work into validated steps:

```yaml
checkpoints:
  - id: inspect-selector
    task: Which `key=value` pair connects the Service to its Pods?
    answer:
      accepted: [app=web]
      caseInsensitive: true
  - id: restore-endpoints
    task: Restore the selector, wait for both endpoints, then run `dojo check`.
    grading:
      all:
        - type: serviceHasEndpoints
          namespace: shop
          name: web
          minEndpoints: 2
```

An answer checkpoint collects one small factual observation. Configuration
checkpoints use the normal state graders. The learner advances with
`dojo check` or `dojo check <answer>`; final grading runs after the last step.

## Variants

Variants stop a lab from training recall. Same task text, same grading,
different cause:

```yaml
variants:
  options:
    - id: selector-mismatch
      faults: [...]
    - id: numeric-target-port
      faults: [...]
    - id: unknown-named-port
      faults: [...]
```

The choice is derived from the attempt's seed, so it is reproducible:

```bash
dojo start services-no-endpoints --seed 9182731
dojo start services-no-endpoints --variant selector-mismatch   # spoils it
```

A variant may override `grading` and `hints` when the diagnosis genuinely
differs — see `node-not-ready`, where stopping the kubelet and stopping
containerd produce the same headline symptom but need different checks.

## Fault types

Run `dojo content types` for the current list.

### Kubernetes

| Type | Fields | Notes |
|---|---|---|
| `kubernetesApply` | `manifest` | Applies a file from the lab directory. Repair deletes it. |
| `kubernetesPatch` | `resource`, `namespace`, `patch`, `patchType` | The workhorse. Repair is a no-op; reset recreates the baseline. |
| `kubernetesDelete` | `resource`, `namespace` | |
| `kubernetesScale` | `resource`, `namespace`, `replicas` | |

### Nodes

| Type | Fields | Notes |
|---|---|---|
| `systemdStop` | `node`, `unit`, `disable` | `disable: true` stops a reboot from quietly fixing the lab. |
| `fileReplace` | `node`, `path`, `source` or `content`, `restart` | Backs up the original so repair can restore it. |
| `nodeExec` | `node`, `script`, `undo` | Escape hatch. `undo` is required. |

## Grader types

### Kubernetes

| Type | Checks |
|---|---|
| `deploymentAvailable` | `namespace`, `name`, `minReplicas` — updated and available replicas, so stale rollout replicas do not pass |
| `podScheduled` | `namespace`, `name`, `scheduled`, `node` — scheduler binding, independent of readiness |
| `serviceHasEndpoints` | `namespace`, `name`, `port`, `minEndpoints` — ready EndpointSlice addresses |
| `httpService` | `namespace`, `service`, `port`, `path`, `expectStatus` — a real request from inside the cluster |
| `authCanI` | `user` or `serviceAccount`, `namespace`, `verb`, `resource`, `expect` — asks the API server |
| `objectExists` | `kind`, `name`, `namespace`, `absent` |
| `jsonPath` | `kind`, `name`, `namespace`, `path` + a match |

### Nodes

| Type | Checks |
|---|---|
| `nodeService` | `node`, `unit`, `state`, `enabled` |
| `nodeReady` | `node`, `ready`, `schedulable` |
| `nodeFile` | `node`, `path`, `absent` + a match |
| `command` | `target`, `command`, `exitCode`, `stdout`, `description` |

### Matches

`jsonPath`, `nodeFile` and `command.stdout` share one comparison shape:

```yaml
equals: "v1.35.8"      # exact, whitespace-trimmed
notEquals: "v1.34.0"
contains: "Ready"
matches: '^v1\.35\.\d+$'
# omit all four to mean "non-empty"
```

`command` is the escape hatch. Give it a `description`, or the requirement
line printed by `dojo grade` will leak the answer:

```yaml
- type: command
  target: cp1
  description: the control plane runs the expected Kubernetes version
  command: kubeadm version -o short
  stdout:
    equals: v1.35.8
```

## Two traps in grading commands

Both of these were found by running the labs, not by reading them, and both
produce a `command` grader that passes when the cluster is broken.

**Do not judge a DNS probe by its exit code.** busybox `nslookup` exits 0 on an
empty `NOERROR` reply as well as on a real answer, and non-zero only on
`NXDOMAIN`. Against deliberately broken cluster DNS it reported success for
5 of 15 queries. Judge the answer instead:

```yaml
# wrong: passes about a third of the time on a cluster that resolves nothing
command: kubectl -n NS exec deploy/probe -- nslookup web >/dev/null

# right: a resolved name produces a `Name:` line
command: |
  kubectl -n NS exec deploy/probe -- nslookup web 2>/dev/null | grep -E '^Name:'
```

A reverse lookup prints `name = ` rather than `Name:`. The same caution
applies to any CLI whose exit code reports "I got a reply" rather than "I got
the reply you wanted" -- `etcdctl snapshot status` on etcd 3.6 prints usage and
exits **zero** for a subcommand that no longer exists, which makes
`a || fallback` silently capture help text.

**A grader that tests a restricted identity must be given nowhere to fall
back to.** `command` graders run as root on the control-plane node, which has
an admin kubeconfig at `/root/.kube/config`. Unsetting `KUBECONFIG` is not
enough — kubectl finds that file, authenticates as cluster-admin, and the
check passes for any credential or none:

```yaml
# wrong: passes with an empty token, and with no ServiceAccount at all
command: |
  env -u KUBECONFIG kubectl --server=https://127.0.0.1:6443 \
    --insecure-skip-tls-verify --token="$token" -n NS get pods

# right: an empty kubeconfig leaves the token as the only way in
command: |
  env -u KUBECONFIG kubectl --kubeconfig=/dev/null \
    --server=https://127.0.0.1:6443 --insecure-skip-tls-verify \
    --token="$token" -n NS get pods
```

An explicit `--kubeconfig <file>` is authoritative and needs no such care. And
assert a **negative** alongside the positive — something the restricted
identity must *not* be able to do. A privilege check that only ever tests what
should succeed cannot tell the intended identity from an admin fallback.

**A fault must not return until its symptom is stable.** `kubectl rollout
status` tells you the new Pods are available, not that they are all serving the
new configuration. CoreDNS runs two replicas behind one Service, so after a
Corefile change roughly half of all queries are answered correctly for a
while: a fault that returned there handed the learner a working cluster that
broke a minute into the attempt. Waiting for the *first* failed probe is not
enough for the same reason. Wait for consecutive failures:

```bash
fails=0
for i in $(seq 1 90); do
  if probe; then fails=0; else
    fails=$((fails + 1)); [ "$fails" -ge 5 ] && break
  fi
  sleep 2
done
```

## Hints

Four levels, escalating, each one a nudge rather than a step:

1. **Area** — which concept is involved
2. **Object** — which object to look at
3. **Approach** — how to compare two things and see the mismatch
4. **Commands** — the actual invocations

Hints are counted. A pass that needed hints does not count toward mastery,
which is the whole point of recording them.

## solution.md

Write it as reasoning, not as a recipe. The structure that works:

- **Working it out** — the diagnostic path, and how to tell the variants apart
- **Fixing it** — two or three equally valid routes, since grading accepts any
- **Checking before you grade** — how to confirm it yourself

## Adding a module

```
curriculum/cka-2026/modules/09-storage/
├── module.yaml
├── lesson.md
└── labs/
```

```yaml
# module.yaml
id: 09-storage
name: Storage
summary: >-
  PersistentVolumes, claims, and why a claim stays Pending.
domains: [storage]
labs:                    # optional; fixes teaching order
  - pvc-pending
```

Then list the module id under `modules:` in `curriculum.yaml`, and add any new
skills to the `skills:` list there.

## Before you commit

```bash
make check
```

Then turn on dev mode, once per checkout, before running anything:

```bash
make dogfood
```

That drops a `.dojo-dev` marker so every attempt from this directory records
to `~/.cka-dojo/progress-dev.json`. Without it, dogfooding writes the same
attempt record a learner's own work writes, and `dojo recommend`, `dojo learn`
and `dojo readiness` cannot tell the two apart: a synthetic pass demotes the
exercise in the ranking and unlocks everything behind it. `DOJO_DEV=1` in the
environment does the same thing for one command, and `make study` removes the
marker. Every command in dev mode prints a banner saying so.

Then actually run it — twice, and with `--variant` for each variant:

```bash
dojo start <lab> && dojo grade      # should fail
# fix it by hand
dojo grade                          # should pass
dojo reset && dojo grade            # should fail again
```

A lab that has not been through that loop is not finished. The engine cannot
tell you that a fault fails to produce the symptom the task describes, or that
a task is ambiguous — only running it can.
