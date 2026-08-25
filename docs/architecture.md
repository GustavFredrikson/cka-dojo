# Architecture

## The four layers

```
┌──────────────────────────────────────────────────────────┐
│  Content            curriculum/ and environments/        │
│                     YAML + Markdown, no Go               │
├──────────────────────────────────────────────────────────┤
│  Engine             internal/{lab,fault,grader,          │
│                     curriculum,progress,cli}             │
├──────────────────────────────────────────────────────────┤
│  Provider           internal/provider (+ lima/)          │
├──────────────────────────────────────────────────────────┤
│  Machines           terminal | cp1 | worker1 | worker2   │
└──────────────────────────────────────────────────────────┘
```

The rule the layout enforces: **adding curriculum never requires Go, and
adding a provider never touches curriculum**.

## Packages

| Package | Responsibility |
|---|---|
| `internal/config` | `~/.cka-dojo`: config, active-lab state, the CLI lock |
| `internal/content` | Where content is read from: a directory, or embedded |
| `internal/provider` | The machine interface: create, exec, copy, shell |
| `internal/provider/lima` | The only implementation today |
| `internal/provider/fake` | In-memory provider so engine tests need no VM |
| `internal/environment` | Profiles, and the orchestration that provisions them |
| `internal/spec` | The lazy `type:`-discriminated YAML envelope |
| `internal/fault` | The vocabulary for breaking a cluster |
| `internal/grader` | The vocabulary for deciding it is fixed |
| `internal/lab` | Lab schema, loader, and the runner that drives an attempt |
| `internal/curriculum` | Domains, skills, modules, lab discovery |
| `internal/learning` | Learning-stage prerequisites and path status |
| `internal/progress` | Attempt history and the mastery rule |
| `internal/recommend` | Transparent weight × gap × recency × stage ranking |
| `internal/ui` | The only writer to the terminal; mirrors everything to a log |

## Why a separate workstation VM

The learner's shell lives on `terminal`, which runs no kubelet and is not a
Kubernetes node. Without that, "troubleshoot the broken control plane" is
awkward: the shell you are working from is on the thing that is broken.

It also means SSH is part of the exercise, the way it is on the exam:

```
student@terminal:~$ ssh worker1
student@worker1:~$ systemctl status kubelet
```

## Why we own kubeadm

Lima ships a `k8s` template that hands you a finished cluster. We do not use
it. Labs need to break the *installation* — stop containerd, corrupt a static
Pod manifest, hold a package at the wrong version — and that requires owning
the install: containerd, the apt pins, `kubeadm init`, the CNI.

The cost is a longer first build. The benefit is that modules 10–12 of the
curriculum are possible at all.

## Provider interface

```go
type Provider interface {
    Name() string
    Available() error
    EnsureNode(ctx, NodeSpec) error
    StartNode(ctx, name) error
    StopNode(ctx, name) error
    DestroyNode(ctx, name) error
    Status(ctx, name) (Status, error)
    List(ctx, prefix) ([]NodeInfo, error)
    Exec(ctx, name, ExecOptions) (ExecResult, error)
    WriteFile(ctx, name, path, content, mode) error
    ReadFile(ctx, name, path) ([]byte, error)
    Shell(ctx, name, user, args) error
}
```

It is **node-granular** on purpose. Environment-level operations
(`Up`, `Stop`, `Destroy`) live in `internal/environment`, composed from these,
so a second provider implements eleven small methods and nothing else.

Two details worth knowing:

- **Scripts travel on stdin**, never in argv. That removes every question
  about how `limactl` and `ssh` reassemble a command line, and means a fault
  script can contain any characters at all.
- **`WriteFile` uses a base64 heredoc** rather than a copy command, so it
  works for root-owned paths without a staging dance.

## Lima specifics that matter

**Node addresses are discovered by subnet, never by "the primary interface".**
The interface layout depends on how Lima is configured: the default user-mode
network gives *every* VM the same address (`192.168.5.15`), while the `user-v2`
network we use replaces that NIC and hands out distinct addresses on
`192.168.104.0/24`. Trusting "the primary IP" would silently register every
node identically under the first layout. So:

1. The engine matches the profile's `network.subnet` when discovering each
   node's address.
2. `--node-ip` is set explicitly in `/etc/default/kubelet` on every node, and
   Calico is told to autodetect within the same CIDR — so the cluster is
   correct regardless of which layout the provider gives us.

**Name resolution is ours.** The engine writes an `/etc/hosts` block on every
node rather than relying on provider DNS, so `ssh worker1` and Kubernetes node
names agree.

**First-run creation is serialised.** Several concurrent `limactl start` calls
each try to generate Lima's shared SSH key; the losers hit an interactive
"overwrite?" prompt and die. Creation runs in parallel again as soon as the key
exists.

## How a lab attempt flows

```
dojo start <lab>
    │
    ├── load curriculum, find the lab
    ├── enforce pass/mastery prerequisites unless explicitly skipped
    ├── ensure the environment is up (build it if not)
    ├── pick a variant from the seed
    ├── build a Plan: manifests + faults + checks
    │
    ├── Runner.Setup:
    │      apply baseline manifests
    │      wait for the baseline to settle    <- so the learner does not
    │      inject faults                         diagnose a slow start-up
    │
    └── write state.json, print task.md

dojo check                         <- interactive exercises only
    │
    ├── validate a short answer and/or current cluster state
    └── persist the next checkpoint

dojo grade
    │
    ├── rebuild the Plan from content and state
    ├── run every check (all of them, even after the first failure)
    └── record the result against the current attempt
```

State on disk holds only identifiers — lab id, variant, seed, start time. The
plan is rebuilt from content each time, so editing a lab and re-grading works
while authoring.

## Where the answers live

The host holds `solution.md`, the grader definitions and the fault
definitions. The guest receives **only rendered task text**. Manifests are
staged into `/run/dojo` on the control plane, applied, and deleted immediately.

A learner with `sudo` on `cp1` could in principle catch a staged file in the
window before deletion. That is an accepted limit for a self-study tool; what
matters is that nothing is *sitting there* to stumble across.

## Reset has two levels

**`dojo reset`** repairs the faults, deletes the baseline objects, waits for
namespaces to finish terminating, and rebuilds the scenario. Fast, and right
for the normal case.

**`dojo env reset`** destroys and rebuilds the machines. This is the escape
hatch, and it exists because making soft reset undo *every* possible learner
action is a bottomless engineering problem — `kubectl delete ns kube-system`
is not something a lab can anticipate.

There are deliberately no VM snapshots. Snapshot behaviour varies across
Lima, vz, QEMU and storage backends, and chasing that is infrastructure work
unrelated to learning Kubernetes.
