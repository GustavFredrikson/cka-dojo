# cka-dojo roadmap

Living document. **Update it in the same commit as the work it describes.** Any
agent or human picking this repo up should be able to read this file alone and
know what exists, what is next, and which decisions are already settled.

- Status legend: `done` / `in progress` / `next` / `later`
- Last reviewed: 2026-09-03

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
| local-path-provisioner | `v0.0.37` (non-default class) | `environments/standard/environment.yaml` |
| ingress-nginx | `controller-v1.15.1` (NodePort) | `environments/standard/environment.yaml` |
| Gateway API | `v1.6.1` standard channel | `environments/standard/environment.yaml` |
| NGINX Gateway Fabric | `v2.7.0` (NodePort) | `environments/standard/environment.yaml` |
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
- [x] Close the competency gaps found by auditing the content against the
  published 2025-02-18 revision: `kubectl top` and resource-usage diagnosis
  (`14-observability`), reclaim policies, ResourceQuota and LimitRange
  (`10-admission`), Pod anti-affinity and topology spread
- [ ] Add `dojo placement` once enough low-stage exercises exist to produce an
  honest per-skill result

Current size: 62 dogfooded exercises across thirteen modules.

**Known competency gaps, in priority order.** These are bullets in the
published curriculum with no exercise behind them:

1. kubeadm cluster creation, cluster lifecycle/upgrade and an HA control
   plane — these wait on the milestone 5 and 7 environments.
2. LoadBalancer and ExternalName Service types.
3. CoreDNS configuration: the DNS labs use CoreDNS but never repair it.
4. Extension interfaces (CNI/CSI/CRI) are mentioned but never exercised.

That list is ordered by curriculum bullet. Section 4.1 reorders the same
ground by expected exam yield and adds etcd backup and restore, which is
missing from it entirely and is not blocked on a new environment profile.

Ingress, the Gateway API and dynamic provisioning are now covered, which
closed the two largest holes. Domain balance still leans away from the
published weighting: cluster architecture remains under-represented against
its 25% target, and closing gap 1 is what corrects it.

### Milestone 5 - special environments - `later`

- [ ] `raw` profile (bare Linux nodes: containerd, kubeadm init/join, CNI labs)
- [ ] `upgrade-1.34` profile (1.34.x -> 1.35.x kubeadm upgrade drill)
- [ ] Profile switching keeps stopped VMs on disk (`dojo env list/prune`)

### Milestone 6 - exam mode - `later`

Weighted task selection, `conflicts:` detection between labs, wall-clock
120-minute timer persisted to disk (never tied to a running process), hidden
grading until `dojo exam finish`, scoring report.

Section 4.3 argues this should be taken **before** milestones 5 and 7 — it is
the largest realism lever in the repo and the only one of the three that needs
no new environment profile — and lists four further requirements.

### Milestone 7 - HA - `later`

`ha` profile: cp1/cp2/cp3 + worker + an API endpoint, covering the
"highly-available control plane" competency.

## 4. Exam realism

Content audit, 2026-09-03. The question this section exists to answer: **if a
learner masters everything currently in the dojo, are they ready to sit the
exam?**

Honest answer today: ready on Troubleshooting, Services and Networking,
Storage and Workloads; exposed on Cluster Architecture; and never tested under
time pressure. Per-task difficulty sits at roughly real-exam level and clearly
below Killer.sh, for three separable reasons — what is covered, how tasks are
shaped, and how they are delivered. One subsection each, plus a grading-fidelity
defect found during the same audit.

Ordering note: 4.1 outranks 4.2 and 4.3 combined. A topic that has never been
practised scores zero however well it is presented.

### 4.1 Content gaps that cost real marks - `next`

The gap list in milestone 4 is organised by curriculum bullet. This one is
organised by expected exam yield, and it opens with an item that list does not
mention at all:

- [ ] **etcd backup and restore.** Near-certain on the real exam, absent here,
  and — unlike the rest of gap 5 — **not blocked on a new environment
  profile**. cp1 already runs a stacked etcd with its client certificates on
  disk. A lab needs a `nodeExec` fault and `nodeFile` plus `command` graders;
  nothing new in the engine. Worth three variants: take a snapshot to a given
  path; restore a supplied snapshot into a fresh `--data-dir` and repoint the
  static Pod; diagnose a control plane that is down *because* a restore was
  done wrongly. The highest-yield single lab the repo can add today.
- [ ] Certificate and kubeconfig repair: an expired client certificate, a
  kubeconfig pointing at the wrong server or carrying the wrong CA,
  `kubeadm certs check-expiration`. Belongs in `12-control-plane`.
- [ ] CoreDNS repair (existing gap 4). The DNS labs consume CoreDNS but never
  break its Corefile or scale the Deployment to zero.
- [ ] LoadBalancer and ExternalName Service types (existing gap 3).
- [ ] Workload primitives with no exercise anywhere: DaemonSet (it appears only
  as `--ignore-daemonsets` in `node-drain`), StatefulSet, Job and CronJob,
  PriorityClass and preemption, init and sidecar containers, `kubectl debug`
  and ephemeral containers.

Closing these also corrects the domain balance milestone 4 flags: every bullet
but the last tags `cluster-architecture` or `troubleshooting`.

### 4.2 Task shape - exam-shaped, not concept-shaped - `next`

Every current lab is atomic: one concept, one outcome, `targetMinutes` 5-12.
Real exam questions bundle three or four actions, and Killer.sh bundles more
while specifying less. A learner who has only ever done atomic tasks has never
practised the thing that actually burns the clock — holding a multi-part
requirement in their head while working.

- [ ] Define a **composite** shape at the `exam` stage: two to four graded
  outcomes spanning at least two skills, `targetMinutes` 12-18. The schema
  already allows it (`grading.all` and `skills` are both lists); what is
  missing is the editorial decision to write them. Buildable from parts that
  already exist — create a Deployment *and* expose it *and* keep it inside an
  existing ResourceQuota.
- [ ] Require a context or namespace switch in `exam`-stage task text. Every
  real question opens with one, and forgetting it is a common way to lose a
  question that was otherwise answered correctly.
- [ ] Stop restating hint content in the task. `workloads-exam-build` describes
  "the standard hostname label" and then hands over `kubernetes.io/hostname` in
  hint 2, so taking the hint costs a mastery flag and buys nothing. At the
  `exam` stage a hint should be a real concession.
- [ ] Reconsider `difficulty: 3` as the ceiling. Nothing in the repo is rated
  above it, so the scale has no headroom for the composite labs above. Either
  widen it to 5 or document 3 as "as hard as the exam gets".

### 4.3 Exam conditions - promote milestone 6 ahead of 5 and 7

Milestone 6 is recorded as `later`. It is the largest single realism lever in
the repo and should be taken first: milestones 5 and 7 both need new
environment profiles, and this needs no VM work at all. Killer.sh is hard
mostly through *delivery* — a wall clock, no feedback, no hints — and none of
that is content.

Beyond the milestone 6 bullets already recorded:

- [ ] Pass mark and a per-domain score report at `dojo exam finish`, so the
  output answers "would I have passed" and "where did I lose it", not just
  "12 of 17".
- [ ] `dojo hint` and `dojo solution` refuse to run inside an exam session.
- [ ] Flag-for-review and skip. Triage — spotting the question that will eat
  fifteen minutes and coming back to it — is an examinable skill, and it
  cannot be practised without the ability to defer.
- [ ] Per-task elapsed time in the report, against each lab's `targetMinutes`.
  The learner needs to know *which* tasks made them slow, not only that they
  ran out.

### 4.4 Grading fidelity - `next`

- [ ] `jsonPath` compares rendered strings, so `equals: 64Mi` rejects
  `65536Ki` and `equals: 250m` rejects `0.25`. Those describe identical
  cluster state, and rejecting them contradicts settled decision 4. The grader
  should compare quantities as quantities whenever both sides parse as one.
  Ten assertions are affected today, across `workloads-build`,
  `workloads-exam-build` and `limitrange-build`.
- [ ] While there: an `equalsAny` form, for the cases where several literal
  spellings are correct and quantity parsing does not apply.

---

## 5. Deliberate non-goals (v1)

Browser UI, SaaS/hosted mode, user accounts, cloud clusters, third-party lab
plugins, MCP/AI integration, gamification, non-macOS hosts, replicating the PSI
exam UI.

## 6. Settled design decisions

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

## 7. Known gaps / risks

- Only macOS/arm64 + Lima is exercised. Nothing else is claimed to work.
- Soft reset cannot undo arbitrary learner damage (e.g. `kubectl delete ns
  kube-system`); `dojo env reset` is the documented answer.
- A determined learner with `sudo` on cp1 could inspect staged fault artefacts
  in the window before they are deleted. Acceptable for a self-study tool.
- PR CI (`.github/workflows/ci.yml`) runs gofmt, `go vet`, `go test` and
  `content validate` on Linux, against the fake provider. Nothing in CI boots a
  VM, so provisioning regressions are only caught by running `dojo setup`
  locally. An integration job that builds a real cluster is still unwritten.

## 8. Verification log

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
2026-09-03, macOS arm64, same cluster:

- All eight new exercises passed the full loop live — start, grade fails for
  the intended reason, hand-fix, grade passes, reset, grade fails again — on
  Kubernetes 1.35.8. Both documented repair routes were exercised separately
  for `quota-blocked` (declare resources / add a LimitRange plus a rollout
  restart) and for `scheduling-spread-pending` (preferred anti-affinity /
  topology spread).
- Live running found and fixed two engine defects and two content defects:
  - `deploymentAvailable` split two jsonpath values on whitespace. An absent
    `.status.availableReplicas` renders as empty, so the *updated* count was
    read as the *available* count: a Deployment with nothing running reported
    "2 available, 0 updated" and pointed the learner at a rollout rather than
    at the scheduler. Both this and `nodeReady` now use an explicit separator.
  - `objectExists` and `jsonPath` printed "in default" for cluster-scoped
    kinds, so a PersistentVolume check named a namespace it has nothing to do
    with. The location is now omitted when the lab gives no namespace.
  - `storage-reclaim` originally had the learner create the PersistentVolume.
    Being cluster-scoped it survived teardown, so the lab passed instantly
    after a reset. The volume now ships in the baseline with the `Delete`
    policy.
  - `scheduling-spread-pending` originally used self-anti-affinity. The rule
    is symmetric, so healthy old Pods repelled the corrected ones and a
    correct fix deadlocked the rollout under any `maxUnavailable` below 2 —
    and pinning a strategy in the baseline did not help, because a learner who
    re-applies a manifest prunes it. It now repels a separate `cache`
    workload, applied through a `kubernetesApply` fault so that cache settles
    on both workers before checkout exists.

2026-09-03, macOS arm64, same cluster (second pass):

- Added four pinned platform addons and the six exercises that need them:
  `08-ingress` (Ingress follow/build/inspect, a 503 repair, and a Gateway API
  build) plus `storage-dynamic`. All six passed the full live loop.
- `dojo setup` installs the addons into an existing cluster without touching
  the rest: markers are per-script, so only the two new steps ran, and a
  second invocation finished in 11s.
- Three real installation defects, each found only by running it:
  - NGINX Gateway Fabric's deploy bundle carries NginxGateway and NginxProxy
    *instances* but not their CRDs, so it must be preceded by `deploy/crds.yaml`.
  - That CRD bundle cannot be applied client-side: the NginxProxy schema alone
    exceeds the 262144-byte `last-applied-configuration` annotation limit, so
    it needs `--server-side`.
  - NGF v2 names its control plane `nginx-gateway`, not `ngf-nginx-gateway-fabric`,
    and provisions a data plane per Gateway rather than up front.
- `ingress-inspect` claimed `/api/` worked under an `Exact` rule. It does not:
  `/api` matches, nginx 301s to `/api/`, and the redirect target no longer
  matches its own rule. The exercise now teaches that, having been corrected
  against observed output.
- `storage-dynamic` first had the learner create the StorageClass, which is
  cluster-scoped and so survived teardown -- the same hazard as
  `storage-reclaim`. It now ships in the baseline as
  `kubernetes.io/no-provisioner`, which also exposes that `provisioner` and
  `volumeBindingMode` are immutable and force a delete-and-recreate.
- local-path is installed deliberately as a non-default StorageClass, so the
  exercises that depend on a claim staying Pending still behave.

- The five-stage Scheduling path passed live end to end. Selector and taint
  variants both produced genuine Pending Pods, accepted distinct valid fixes,
  and reset correctly. Teardown removed every training namespace and restored
  all worker labels and taints; all three Kubernetes nodes remained Ready.
