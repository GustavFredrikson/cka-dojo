# cka-dojo roadmap

Living document. **Update it in the same commit as the work it describes.** Any
agent or human picking this repo up should be able to read this file alone and
know what exists, what is next, and which decisions are already settled.

- Status legend: `done` / `in progress` / `next` / `later`
- Last reviewed: 2026-08-25

---

## 1. What this is

A self-hosted, disposable CKA training platform. A Go CLI (`dojo`) provisions
real kubeadm Kubernetes clusters in Lima VMs, injects faults, and grades the
learner on **resulting cluster state** rather than on typed commands.

Four layers, engine strictly separated from content:

```
CKA curriculum (YAML + Markdown)   <- content, versioned per exam revision
Dojo engine (Go)                   <- lab runner, faults, graders, progress
Environment provider (Lima)        <- VM lifecycle, exec, copy
Linux VMs                          <- terminal | control plane | workers
```

## 2. Pinned versions

These are deliberate pins, not "latest". The exam tracks a Kubernetes minor
that lags upstream, so **never** follow upstream automatically.

| Thing | Pin | Where |
|---|---|---|
| Exam profile | CKA, curriculum revision 2025-02-18 | `curriculum/cka-2026/curriculum.yaml` |
| Kubernetes | `1.35.8` (latest patch of exam minor 1.35) | `environments/*/environment.yaml` |
| Calico | `v3.32.1` (3.32 line is tested against k8s 1.34-1.36) | `environments/*/environment.yaml` |
| metrics-server | `v0.9.0` | `environments/standard/environment.yaml` |
| Guest OS | Ubuntu 24.04 LTS (Lima `_images/ubuntu-24.04`) | `internal/provider/lima` |
| Lima network | `user-v2`, `192.168.104.0/24` | `environments/*/environment.yaml` |

Bumping the exam to a new minor should touch **only** those YAML files.

## 3. Milestones

### Milestone 1 - foundation - `done`

Go CLI, config/state, provider interface, Lima provider, `doctor`, environment
profiles, `standard` provisioning, `setup`, `shell`, `env` subcommands.

Success criterion: `dojo setup && dojo shell` lands you in a working kubeadm
cluster.

### Milestone 2 - lab engine - `done`

Lab schema + loader, `start` / `task` / `reset` / `grade` / `hint` / `solution`,
fault primitives, grader framework, three reference labs, `content validate`.

Implemented fault primitives: `kubernetesApply`, `kubernetesPatch`,
`kubernetesDelete`, `kubernetesScale`, `systemdStop`, `fileReplace`, `nodeExec`.
Implemented graders: `deploymentAvailable`, `podScheduled`,
`serviceHasEndpoints`, `httpService`, `authCanI`, `nodeReady`, `nodeService`,
`nodeFile`, `command`, `objectExists`, `jsonPath`.

### Milestone 3 - learning UX - `done`

- [x] `dojo learn <module>` renders `lesson.md` with an interactive pager
- [x] `dojo learn` shows per-module and overall attempted/passed/mastered totals
- [x] Lessons for all current modules, with a validated common structure
- [x] `dojo progress` skill table and `--by-lab` attempt history
- [x] `dojo recommend` = exam weight x lack of mastery x recency
- [x] `dojo tutor-context` with an explicit metadata allow-list; tests prove it
  excludes variant and seed, and it never receives faults, graders or solutions
- [x] Mastery rule: >=2 passes AND latest pass used 0 hints
- [x] Recoverable `dojo progress reset` and a normal `make install`
- [x] Seven-level learning-stage metadata, separate from technical difficulty
- [x] Ordered paths with pass/mastery prerequisites and an explicit jump-ahead
- [x] Interactive state/answer checkpoints through `dojo check`
- [x] First Services slice: Follow → Build → Inspect → guided repair →
  contextual troubleshooting
- [x] Stage-aware recommendations that never select locked exercises

### Milestone 4 - curriculum expansion - `in progress`

Grow to 45-55 exercises, driven by gaps found while actually studying. Do **not**
mass-generate labs; each one gets dogfooded. Target the published domain
weighting: troubleshooting 30, cluster architecture 25, networking 20,
workloads 15, storage 10.

- [x] Live-dogfood the complete Services path and every contextual variant
- [x] Scheduling path: Follow → Build → Inspect → guided repair → contextual
  troubleshooting, including clean restoration of node labels and taints
- [x] Workloads path: command vocabulary, configuration, inspection, rollouts,
  guided CrashLoop repair and three contextual failure variants
- [x] Expand RBAC into the same progression, with positive and negative
  authorization checks
- [x] Add the Storage progression (PV, PVC, StorageClass, binding, Pending)
- [x] Add NetworkPolicy from default deny through contextual source, port and
  destination failures
- [x] Add an exam-realistic operations batch: HPA, probes, NodePort, DNS,
  static Pods, scheduler recovery, node drain, logs, Helm, Kustomize and CRDs
- [ ] Add `dojo placement` once enough low-stage exercises exist to produce an
  honest per-skill result

Current size: 48 dogfooded exercises across ten modules. Next curriculum slice:
Ingress/Gateway and dynamic provisioning, followed by cluster lifecycle work
that uses the special environments in milestone 5.

### Milestone 5 - special environments - `later`

- [ ] `raw` profile (bare Linux nodes: containerd, kubeadm init/join, CNI labs)
- [ ] `upgrade-1.34` profile (1.34.x -> 1.35.x kubeadm upgrade drill)
- [ ] Profile switching keeps stopped VMs on disk (`dojo env list/prune`)

### Milestone 6 - exam mode - `later`

Weighted task selection, `conflicts:` detection between labs, wall-clock
120-minute timer persisted to disk (never tied to a running process), hidden
grading until `dojo exam finish`, scoring report.

### Milestone 7 - HA - `later`

`ha` profile: cp1/cp2/cp3 + worker + an API endpoint, covering the
"highly-available control plane" competency.

## 4. Deliberate non-goals (v1)

Browser UI, SaaS/hosted mode, user accounts, cloud clusters, third-party lab
plugins, MCP/AI integration, gamification, non-macOS hosts, replicating the PSI
exam UI.

## 5. Settled design decisions

1. **Go, not shell.** Shell only runs *inside* guests (provisioning, faults).
2. **Separate `terminal` VM.** The learner never shells in from a cluster node,
   so "the control plane is broken" labs stay realistic.
3. **We own kubeadm**, rather than using Lima's finished `k8s` template - labs
   need to break the installation itself.
4. **Grade state, not commands.** Any route to the correct state passes.
5. **Content never enters the guest.** Only rendered task text does; manifests
   are staged to cp1, applied, then deleted. Solutions/graders stay on the host.
6. **Addresses are discovered by subnet, and `--node-ip` is explicit.** Lima's
   interface layout depends on configuration - the default user-mode network
   gives every VM the same address (192.168.5.15), while `user-v2` replaces
   that NIC with a distinct 192.168.104.x. Matching `network.subnet` and
   pinning `--node-ip` is correct under either.
7. **No VM snapshots.** Soft reset (re-apply baseline + re-inject fault) for the
   normal case, `dojo env reset` as the guaranteed escape hatch.
8. **Provider is node-granular** (`EnsureNode`, `Exec`, `CopyTo`, ...) and
   `internal/environment` composes it into environment-level operations. Adding
   a second provider must not touch curriculum code.
9. **Content is embedded but overridable** via `DOJO_CONTENT` / `--content`, so
   a colleague needs one binary while we develop against the repo.

## 6. Known gaps / risks

- Only macOS/arm64 + Lima is exercised. Nothing else is claimed to work.
- Soft reset cannot undo arbitrary learner damage (e.g. `kubectl delete ns
  kube-system`); `dojo env reset` is the documented answer.
- A determined learner with `sudo` on cp1 could inspect staged fault artefacts
  in the window before they are deleted. Acceptable for a self-study tool.
- PR CI (`.github/workflows/ci.yml`) runs gofmt, `go vet`, `go test` and
  `content validate` on Linux, against the fake provider. Nothing in CI boots a
  VM, so provisioning regressions are only caught by running `dojo setup`
  locally. An integration job that builds a real cluster is still unwritten.

## 7. Verification log

2026-08-25, macOS arm64, Lima 2.2.0:

- Cold `standard` environment build completed in 3m49s after the Ubuntu image
  was available; cp1, worker1 and worker2 registered Ready on Kubernetes 1.35.8.
- `dojo shell` reached the terminal as `student`; SSH to worker1 worked.
- All three `services-no-endpoints` variants failed before repair; a student
  fix passed all object, EndpointSlice and in-cluster HTTP checks; soft reset
  re-injected the fault and immediate stop/start no longer races namespace
  deletion.
- `rbac-namespace-reader` denied the required access before repair and passed
  all positive and negative authorization checks after a Role/RoleBinding fix.
- Both `node-not-ready` variants produced a NotReady worker. Repairing kubelet
  passed; ending the containerd variant restored and re-enabled the service.
- Live testing found and fixed four engine defects: first-run Lima SSH-key
  creation races, EndpointSlice null handling, misplaced probe namespace flags,
  and stdin-consuming guest commands truncating streamed scripts.
- Milestone 3 smoke testing used an isolated `DOJO_HOME`: all lessons rendered,
  recommendation factors were visible, progress reset archived history, and
  tutor context omitted the active variant and seed. The embedded-content
  binary was installed successfully as `~/.local/bin/dojo`.
- The five-stage Services path passed live end to end. All three contextual
  Service variants failed before repair and passed after distinct state-based
  fixes; reset re-injected the selected fault.
- The five-stage Scheduling path passed live end to end. Selector and taint
  variants both produced genuine Pending Pods, accepted distinct valid fixes,
  and reset correctly. Teardown removed every training namespace and restored
  all worker labels and taints; all three Kubernetes nodes remained Ready.
