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
- [x] `dojo readiness`: exam-weighted mastery with hard gates, and the
  curriculum's own declared gaps printed as caveats on the verdict

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
- [x] Close every §4.1 content gap that the `standard` profile can reach: etcd
  backup and restore, kubeconfig repair and certificate inspection, CoreDNS
  repair, LoadBalancer and ExternalName, and the workload primitives
  (DaemonSet, StatefulSet, Job, CronJob, init containers, priority and
  preemption, ephemeral containers) in the new `11-workload-primitives`
- [ ] Add `dojo placement` once enough low-stage exercises exist to produce an
  honest per-skill result

Current size: 81 dogfooded exercises across fourteen modules. The
etcd/CoreDNS/workload-primitives batch went through the live loop on
2026-09-07; §8 records the nine defects that found, two of which were in
content that had already shipped. A second audit the same day found five
further gaps that the first had missed — all reachable on the `standard`
profile — and the exercises closing them went through the loop on 2026-09-08.

- [x] Close the five gaps the §4.1 audit itself missed, found by checking the
  content against the published curriculum bullet by bullet rather than
  against the existing gap list: `rbac-csr` (the CertificateSigningRequest
  workflow, from key to kubeconfig to RBAC), `sa-token` (TokenRequest and
  projected ServiceAccount tokens), `psa-restricted` (Pod Security
  Admission), `sidecar-build` (init containers with `restartPolicy: Always`)
  and `operator-reconcile` (an operator reconciling a hand edit away)

**Known competency gaps.** What is left is what the `standard` profile cannot
reach:

1. kubeadm cluster creation, cluster lifecycle/upgrade and an HA control
   plane — these wait on the milestone 5 and 7 environments.
2. Certificate *expiry*. `certs-expiration` reads the dates and explains
   renewal; `kubeconfig-repair` rebuilds a broken kubeconfig. Nothing makes a
   certificate actually expire, because kubeadm issues them for a year and
   there is no supported way to backdate one. A lab would need a clock skew
   fault on cp1, which breaks far more than it teaches.
3. Extension interfaces (CNI/CSI/CRI). CRI inspection is reachable on
   `standard` today; replacing a CNI or a CSI driver is not.

Domain balance is now much closer to the published weighting: cluster
architecture went from 3 exercises to 8 in `12-control-plane` alone, and
`dojo readiness` reports 19 labs against its 25% target. Closing gap 1 is what
finishes the job.

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

Content audit, 2026-09-03; revised 2026-09-07 after the §4.1 batch landed. The
question this section exists to answer: **if a learner masters everything
currently in the dojo, are they ready to sit the exam?**

Honest answer today: ready on Troubleshooting, Services and Networking,
Storage and Workloads; on Cluster Architecture, covered for etcd,
certificates, kubeconfig, static Pods and node maintenance, and still exposed
on cluster *lifecycle* — creating a cluster, joining a node, upgrading, HA.
That last group is the whole of what is left, and it is blocked on environment
profiles rather than on writing labs, which promotes milestone 5 from "later"
to the next thing that buys coverage.

Still never tested under time pressure. Per-task difficulty sits at roughly
real-exam level and clearly below Killer.sh, for three separable reasons — what
is covered, how tasks are shaped, and how they are delivered. One subsection
each, plus a grading-fidelity defect found during the same audit.

Ordering note: 4.1 outranked 4.2 and 4.3 combined, and is now done. With
content no longer the binding constraint, 4.3 (exam conditions) is the largest
remaining realism lever.

### 4.1 Content gaps that cost real marks - `done`

This list was organised by expected exam yield. Everything on it is written and
has been through the live loop.

- [x] **etcd backup and restore.** Three exercises rather than three variants,
  because the diagnostic paths diverge too far to share task text:
  `etcd-snapshot` (Build — take a snapshot, read the endpoint and certificate
  paths off the static Pod manifest), `etcd-restore` (guided-fix — restore a
  planted snapshot into a fresh `--data-dir` and repoint the manifest) and
  `etcd-unreachable` (Diagnose — an API server that will not start because
  etcd moved its client port, with no `kubectl` available to find out why).
  `etcd-restore` grades a value that exists *only* inside the snapshot, so a
  hand-recreated object cannot pass.

  This needed the one environment change in the batch:
  `etcd-tools.sh` installs `etcdctl` and `etcdutl` on the control plane,
  version read out of kubeadm's own etcd manifest rather than pinned in the
  profile — the client has to match the server's storage format, and kubeadm
  chooses the server. It prefers extracting the binaries from the image
  containerd already has, and falls back to the release tarball.
- [x] Certificate and kubeconfig repair: `certs-expiration` (Inspect — the
  expiry listing, the CA/leaf split, decoding `admin.conf`'s embedded client
  certificate, and why the `O` and not the `CN` is what grants cluster-admin)
  and `kubeconfig-repair` (contextual-fix, three variants: wrong server port,
  untrusted CA, and a `current-context` that does not exist). Grading requires
  the repair to be in `~/.kube/config`, not in `KUBECONFIG`.

  An *expired* certificate is still uncovered and is now recorded as a
  `knownGap` with its reason rather than as a to-do: certificates are issued
  for a year and cannot be backdated.
- [x] CoreDNS repair: `coredns-down` (contextual-fix, two variants — the
  Deployment scaled to zero, and a `kube-dns` Service selecting a label
  nothing carries) and `coredns-corefile` (guided-fix — the `kubernetes`
  plugin serving a different cluster domain than every Pod's `resolv.conf`
  searches, so external names resolve and cluster names do not). Both undo by
  restoring kubeadm's own configuration, because cluster DNS is in no lab's
  baseline and a patch fault could not be reset.
- [x] LoadBalancer and ExternalName Service types: `services-types`, including
  checkpoints on why `EXTERNAL-IP` stays `<pending>` with no cloud controller
  manager and what still works anyway.
- [x] Workload primitives, as the new `11-workload-primitives`:
  `daemonset-build` (the control-plane toleration, and that
  `desiredNumberScheduled` reports 2 of 3 with nothing marked wrong),
  `jobs-build`, `statefulset-build` (headless Service, `volumeClaimTemplates`
  and per-Pod DNS), `initcontainer-repair`, and `priority-preemption`.

Two decisions inside that batch worth recording:

- `priority-preemption` fills the cluster with a **custom extended resource**
  (`dojo.cka/slots`, capacity 2, advertised on worker1 by the fault) rather
  than with CPU or memory. Filling a real resource would depend on whatever
  the platform addons happen to be requesting, and the symptom would drift
  between runs; two slots on one node is two slots on one node.
- `priority-preemption` ships `dojo-critical` in its baseline with a *wrong*
  value, which is also the exercise: a PriorityClass's `value` is immutable,
  so it cannot be patched into shape. That solves the free-pass problem from
  `docs/authoring-labs.md` — a cluster-scoped object the learner created would
  survive reset — by making the object the lab's own.

Also written, for the `kubectl debug` half of the last bullet:
`debug-ephemeral` in `15-troubleshooting`, on a distroless Pod with no shell.
Grading requires the ephemeral container to be present and targeting the right
container, which is what stops the exercise being satisfied by reading the
manifest instead.

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
- [x] Reconsider `difficulty: 3` as the ceiling. Widened: `priority-preemption`
  is the first `difficulty: 4`, and the first `exam`-stage lab outside
  `02-workloads`. It is also the first composite in the sense 4.2 asks for —
  three graded outcomes across `pod-priority`, `preemption` and
  `resource-management`, opening with a namespace switch — so it is worth
  reading as the shape for the rest.

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

### 4.5 Readiness — `done`, with one dependency

`dojo readiness` (settled decision 10) answers "am I ready" from existing
history. Two things still cap what it can claim, both tracked above:

- It cannot see pace under load, only per-lab pace. That needs milestone 6
  (§4.3).
- Its `knownGaps` list is only as honest as 4.1. Five entries were removed
  when that batch landed; the five that remain are three environment profiles,
  certificate expiry, and CNI/CSI replacement. Two of those five are
  permanent-ish caveats rather than to-do items, and their `note` now says so
  — a learner reading the verdict should be able to tell "not written yet"
  from "cannot be simulated here".

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
10. **Readiness names what it cannot see.** `dojo readiness` scores only
    mastered labs, weighted by published domain weight, and an untouched
    domain blocks a ready verdict outright rather than being averaged away.
    Because a score over the labs that happen to exist is not a statement
    about the exam, `curriculum.yaml` declares `knownGaps` — published
    competencies with no exercise behind them — and the report prints them
    with every verdict. Delete a gap entry the day its lab lands.

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
- **A node address that changes breaks the cluster irrecoverably** — now
  detected rather than fixed, and the original diagnosis was wrong.

  Found 2026-09-07: a cluster that had been stopped came back with cp1 on
  `192.168.104.4` when it had been built for `.5`, and `.5` now belonging to
  worker2. Everything kubeadm writes embeds cp1's address — the API server
  certificate's SANs, all four static Pod manifests, every kubeconfig in
  `/etc/kubernetes`, and each kubelet's `--node-ip` — so the control plane
  could not come back. kubelet logged `failed to validate nodeIP: node IP
  "192.168.104.5" not found in the host's network interfaces`, and
  `dojo setup` sat in `WaitReady` for its full ten minutes before failing with
  nothing that named the cause.

  This was first written up here as "a stopped environment does not survive
  being started again… it is a race, not a one-off". **That was wrong**, and
  measuring it on 2026-09-08 said so: stopping all four machines and starting
  them again preserved every address exactly, twice, once in forward order and
  once reversed, and the cluster came back with all three nodes Ready both
  times. Lima derives a `user-v2` address from each instance's MAC, and the MAC
  is stable for the life of an instance — so start order is irrelevant and an
  ordinary stop/start is safe. An address only moves when the instance itself
  is replaced, which is what `dojo env reset` does deliberately, or if the
  network backend's assignment scheme changes underneath a built cluster. The
  original breakage was one of those; which one is no longer recoverable from
  the evidence.

  That reframing changed the fix. Pinning static addresses in the guest would
  harden a path that is not actually broken, and rewiring a live cluster onto
  new addresses — regenerating certificate SANs, rewriting four manifests,
  every kubeconfig and every `--node-ip` — is a large, risky feature for an
  event this rare. What was worth doing is making the failure loud:
  `verifyNodeAddresses` compares each cluster node's current address against
  the one recorded in its own `/etc/default/kubelet` and refuses to continue if
  they differ, naming both addresses and pointing at `dojo env reset`. It
  refuses in under a second instead of hanging for ten minutes, and it is
  silent on a healthy environment.

  Both halves are covered by tests, which was not optional: the first
  implementation of the parser was wrong and the check silently never fired.
  It split `/etc/default/kubelet` on whitespace and looked for a field
  beginning `--node-ip=`, but the file holds one assignment —
  `KUBELET_EXTRA_ARGS=--node-ip=10.0.0.1` — so nothing ever matched and every
  node looked unprovisioned. It passed the healthy case for the wrong reason
  and was only caught by deliberately simulating drift on a live node.
  `TestNodeIPArg` now pins the parsing, `TestUpRefusesWhenANodeAddressMoved`
  proves `Up` aborts before `kubeadm init`, and
  `TestVerifyNodeAddressesAcceptsAHealthyEnvironment` covers agreement and the
  never-provisioned first run. Under the fake provider the check is inert by
  default, so without those tests the new path would have had no coverage at
  all.

  Left undone deliberately: recovery is still a rebuild. If that ever becomes
  a real cost — a `raw` or `ha` profile makes rebuilds much more expensive —
  the rewiring is the thing to build, and this check is what will tell you it
  is needed.

- **A cold boot can leave one `calico-node` Pod stuck in
  `Init:CreateContainerConfigError`.** Seen 2026-09-08 after stopping and
  starting the whole environment twice. The init container reports `services
  have not yet been read at least once, cannot construct envvars` — a kubelet
  start-up race where a Pod needing service environment variables is created
  before kubelet has synced the service list. It does not clear on its own; it
  sat for half an hour. Benign in practice, because the Pod's own container is
  `1/1` and CNI is installed, so pod networking works and the node is Ready —
  but it looks alarming and it makes `kubectl get pods -A` never come clean.
  `kubectl -n calico-system delete pod <name>` and letting the DaemonSet
  replace it fixes it in seconds. Worth teaching rather than fixing: it is a
  good example of a Pod whose status is worse than its actual health.

- **One unreproduced test flake.** `TestBaseProvisionWiresNodesTogether` and
  `TestStepsAreSkippedWhenAlreadyDone` both failed once, together, during a
  `make check` run while four VMs were booting on the same machine. Six
  subsequent runs of `make check` and of the package alone were clean, and
  both tests use an isolated `DOJO_HOME` and their own fake provider, so there
  is no shared state to explain it. Recorded rather than closed: if it recurs,
  the machine being under heavy load is the first thing to suspect.

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

2026-09-07, macOS arm64, Lima 2.2.0, rebuilt `standard` cluster:

All 14 exercises in the §4.1 batch passed the full loop live — start, grade
fails for the intended reason, hand-fix following the exercise's own
`solution.md`, grade passes, reset, grade fails again — on Kubernetes 1.35.8
with etcd 3.6.6. `make check` passes: 14 modules, 76 exercises. The cluster was
left with three nodes Ready, 25 Running Pods, no leftover training namespaces
and no leftover cluster-scoped objects.

**Confirmed as designed.** `etcd-tools.sh` took the image-extraction route in
0s, no download, version-exact against kubeadm's `registry.k8s.io/etcd:3.6.6-0`.
`etcd-restore` recovered a value that exists only inside the snapshot, and
reset re-injected the failure afterwards. `daemonset-build`'s premise held
exactly: a DaemonSet with no toleration reports `DESIRED 2 READY 2` with
nothing anywhere marked wrong. `priority-preemption` worked end to end —
`kubectl patch node --subresource=status` advertised the extended resource,
`value: Forbidden: may not be changed in an update` came back verbatim, the
scheduler preempted for an extended resource, and reset restored
`dojo-critical` to its wrong value, so the cluster-scoped free-pass hazard is
genuinely closed. `statefulset-build` left no PVC or PV behind after reset.

**Nine defects found, all fixed. Two were pre-existing.**

1. **`crictl` was never installed** (pre-existing). `kube-node.sh` wrote
   `/etc/crictl.yaml` and no binary. Every "the API server is down, ask the
   runtime" path was therefore broken — the whole diagnostic route of
   `etcd-unreachable`, and the hints and solutions of `controlplane-scheduler`
   and `node-not-ready`, all of which tell the learner to run `crictl`. Fixed
   by installing `cri-tools` from the same pinned apt repo (v1.35.0).
2. **A node address had moved and broken the cluster** (pre-existing, and the
   reason this session began with a rebuild). Recorded in §7 — where the
   diagnosis was initially overstated as "stop/start breaks the cluster" and
   has since been corrected against measurement, along with the fail-fast
   check that now catches it.
3. **`etcdctl snapshot status` on etcd 3.6 prints the `snapshot` usage text and
   exits *zero*.** The subcommand moved to `etcdutl`, and an unrecognised one
   falls through to help rather than failing. So `etcdutl … || etcdctl …` never
   took the fallback and fed help output to `jq`. `etcd-snapshot` now picks the
   binary by what exists and judges the result by whether it parses; the
   solution teaches the trap, because it is exactly the kind of thing that
   makes a script look like it worked.
4. **busybox `nslookup` exits 0 on an empty NOERROR reply**, not only on a real
   answer — it fails only on NXDOMAIN. Measured against deliberately broken
   cluster DNS, its exit code reported "resolved" for **5 of 15** queries. Every
   `nslookup … >/dev/null && ok` assertion is therefore unsound. This affected
   `coredns-down`, `coredns-corefile` and `statefulset-build` — and
   **`dns-policy`, which was already dogfooded and shipped**. All four now
   require a `Name:` line (or `name = ` for reverse lookups) in the output.
   Worth remembering as a general rule: judge a DNS probe by its answer, never
   by its exit status.
5. **`coredns-corefile` could enshrine a broken Corefile as its own baseline.**
   Its fault saved the ConfigMap unconditionally, so after an attempt was left
   unfixed the next setup captured the *broken* file as the "original" and
   teardown then restored that. This is precisely the hazard documented in
   `docs/authoring-labs.md` — which this batch added, and then walked into two
   files later. The backup is gone: the fault is a single reversible
   substitution, so undo reverses it, which is idempotent and cannot capture
   anything.
6. **Both CoreDNS faults returned before the symptom was stable.** CoreDNS runs
   two replicas behind one Service and a config change reaches them at
   different moments, so for a window after the rollout roughly half of all
   queries were still answered correctly — `dojo start` handed over a cluster
   that worked and broke a minute into the attempt. Waiting for the *first*
   failed lookup was not enough either, for the same reason. Both faults now
   wait for consecutive failures.
7. **`etcd-unreachable`'s etcd readiness check was flaky.** A mirror Pod's
   status is written by kubelet through the API server, so while the API server
   is down it goes stale: `kubectl wait --for=condition=Ready -l component=etcd`
   saw `Pending` for minutes after etcd was serving perfectly. Replaced with
   `etcdctl endpoint health`, which asks etcd rather than asking the thing that
   was broken.
8. **`debug-ephemeral`'s central technique did not work as written.**
   `/proc/1/root/...` returned `Permission denied`: the `pause` image runs as
   uid 65535, the debug container is root without `CAP_SYS_PTRACE`, and reading
   `/proc/<pid>/root` for a process you do not own needs it. `--target` was
   fine — PID 1 was the target's process. The fix is `--profile=sysadmin`, and
   the exercise is better for it: the lesson is now that `kubectl debug`
   profiles decide what you can see, which is a real and examinable detail.
9. **`debug-ephemeral` could burn its own graded name.** The
   `ephemeralContainers` subresource is append-only — a container cannot be
   removed, renamed or reconfigured once attached — so a learner whose first
   attempt used the default profile was permanently unable to make a container
   of that name work. Grading now asks whether *an* ephemeral container targets
   `vault`, and the task warns that the flags are a one-shot decision.

**Quoted output corrected against reality** in six places, all of which had
been written from memory: the API server's etcd failure is a wall of repeating
gRPC `addrConn.createTransport` noise rather than a tidy one-line message; the
`Preempted` event lands on the *victim* and names the winner by UID, not by
namespace/name; `kubectl logs` on an initialising Pod prints a genuinely
helpful `Defaulted container "web" out of: web, wait-for-db (init)` line above
the error; a missing context fails as `Error in configuration: context was not
found for specified context: …`; leaf certificates report `364d` residual, not
365; and `kubeadm certs check-expiration` prints `9y` for the CAs, which is
residual time and not the ten-year validity — a checkpoint that asked "how many
years?" accepted only one of the two defensible readings and now says which it
wants.

2026-09-07, second audit and batch, same cluster:

The §4.1 gap list was organised by expected exam yield and was still
incomplete, because it had been written by reordering the existing
`knownGaps` rather than by re-reading the published curriculum. Checking the
content bullet by bullet found **five competencies with no exercise that were
not recorded as gaps at all**, every one of them reachable on the `standard`
profile:

- CertificateSigningRequests and issuing a user credential — nothing generated
  a key and CSR, approved it, or built a kubeconfig from the result. A
  frequent exam task, and the largest single omission.
- ServiceAccount tokens — ServiceAccounts were used as RBAC subjects
  throughout `04-rbac`, but nothing issued one with `kubectl create token` or
  mounted a projected token.
- Pod Security Admission — `10-admission` covered ResourceQuota and
  LimitRange and nothing else in the admission chain.
- Sidecar containers — explained in the `11-workload-primitives` lesson this
  batch added, exercised nowhere.
- Operators — `crd-resource` created a CRD; nothing exercised a controller
  reconciling a custom resource.

The lesson worth keeping: a gap list maintained by editing itself drifts.
`knownGaps` now has five entries, all genuinely blocked on an environment
profile or on something that cannot be simulated, and the audit that produced
that number was done against the curriculum, not against the previous list.

Three environment facts were verified before designing anything, rather than
assumed: `openssl` 3.0.13 is on the workstation; Pod Security Admission is
active and its rejection names every rule broken; and **Calico here is
operator-managed** — `calico-node` carries an `ownerReference` to
`Installation/default`, a direct patch to its `updateStrategy` is reverted
within 20 seconds, and that revert restarts nothing. That last fact is what
`operator-reconcile` is built on, which means the module gets a real operator
lesson without installing anything or risking pod networking.

**Verification state.** All five passed the full loop live —
`operator-reconcile` on 2026-09-07, the other four on 2026-09-08 after a study
session handed the cluster back. Confirmed as written: the CSR chain end to
end (`Pending` → `Approved,Issued`, and `kubectl auth whoami` through the
built kubeconfig reports `Username: dojo-auditor`, `Groups: [dojo-auditors
system:authenticated]`); a projected token landing at the mounted path with
`aud: ["dojo-metrics"]` and an hour's expiry; busybox running happily under
`restricted` with `runAsUser: 1000`, and the four-rule rejection appearing on
the ReplicaSet exactly as quoted; and a `restartPolicy: Always` init container
reporting `running`, making the Pod read `2/2`, and carrying the app's lines
into `kubectl logs -c log-shipper`.

**Two more defects, both in graders, both only findable live.**

10. **`sa-token`'s token check passed with no ServiceAccount at all** — a
    silent false pass. `command` graders run as root on the control-plane
    node, which has an admin kubeconfig at `/root/.kube/config`, so
    `env -u KUBECONFIG kubectl --token=…` found that file and made the request
    as **cluster-admin**: the check would have passed for any token, or none.
    Fixed with `--kubeconfig=/dev/null`, which leaves the token as the only
    way in, plus a negative assertion (the identity must *not* read Secrets)
    so an admin fallback can never look like success. The general rule is now
    in `docs/authoring-labs.md`: a privilege check that only tests what should
    succeed cannot tell the intended identity from an admin fallback.
11. **`sidecar-build`'s graders were flaky in the one window that matters.**
    `kubectl logs deployment/app -c log-shipper` and
    `-l app=app -o jsonpath='{.items[0]…}'` both select an arbitrary Pod
    carrying the label, and during the learner's own rollout that is often the
    *old* terminating Pod, which has no sidecar — so grading immediately after
    a correct answer failed with `container log-shipper is not valid for pod
    …`. Both now iterate over Running Pods only, and the solution documents
    the trap, because the error reads as "your sidecar is missing" when the
    real answer is "you are looking at the Pod you replaced".

One cleanup found while resetting: `rbac-csr` removed the key material it
tells the learner to create but not the `ca.crt` copied off the control plane,
so teardown left a file behind. Now removed with the rest.
