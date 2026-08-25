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

## The three rules

**1. The task states the symptom, never the cause.**

> The Pods are healthy. Requests sent to the Service do not reach them.

not

> The Service selector does not match the Pod labels.

Diagnosis *is* the exercise. If the task names the broken field, all that is
left is typing.

**2. Grade state, never commands.**

Never ask "did they run `kubectl patch`". Ask "does the Service now have ready
endpoints". Imperative kubectl, `kubectl edit`, a rewritten manifest and a
patch are all correct, and the exam scores them all the same.

**3. Every fault must be undoable.**

`dojo reset` calls each fault's repair. Prefer a named primitive, which knows
how to undo itself. `nodeExec` requires you to supply the `undo` script, and
validation rejects it if you do not.

## lab.yaml

```yaml
schemaVersion: 1
id: services-no-endpoints          # unique across the whole curriculum
title: A Service that does not answer
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
| `deploymentAvailable` | `namespace`, `name`, `minReplicas` — available replicas, not just desired |
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
