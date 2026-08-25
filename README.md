# cka-dojo

CKA practice on real, disposable Kubernetes clusters.

`dojo` builds a four-machine kubeadm cluster in local VMs and teaches each
topic through a deliberate ladder: learn, follow, build, inspect, guided fix,
contextual fix, diagnose, then exam. It grades **the state you leave behind**
rather than the commands you type.

```
$ dojo learn services

Services learning path
·  1  Follow           services-follow
🔒 2  Build            services-build
🔒 3  Inspect          services-inspect
🔒 4  Fix, guided      services-guided-selector-fix
🔒 5  Fix, contextual  services-no-endpoints
```

## Requirements

- macOS on Apple Silicon (the only combination that is exercised today)
- 16 GiB of RAM, 40 GiB of free disk
- [Lima](https://lima-vm.io): `brew install lima`
- Go 1.25+ to build

## Getting started

```bash
make install
dojo doctor
```

By default this installs the self-contained binary to `~/.local/bin/dojo`.
Use `PREFIX=/usr/local make install` to choose a different prefix.

`doctor` checks the host, the dependencies and the content before you spend
twenty minutes provisioning a cluster that was never going to work.

```bash
dojo setup
```

The first build downloads an OS image and installs packages; expect 15–25
minutes. It is idempotent and resumable — interrupt it and run it again.

```bash
dojo shell
```

That drops you onto the workstation as `student`, with `kubectl`, `helm`,
completions and a working kubeconfig. From there `ssh cp1`, `ssh worker1` and
`ssh worker2` all work.

## The loop

```bash
dojo labs                  # what is available
dojo learn services        # concise theory and diagnostic workflow
dojo recommend             # what your history says to practise next
dojo start <lab>           # build the scenario, print the task
dojo shell                 # go and fix it
dojo check [answer]        # validate the current tutorial checkpoint
dojo grade                 # check the cluster against the requirements
dojo hint                  # a nudge, one level at a time, recorded
dojo solution              # a worked answer
dojo reset                 # rebuild this scenario
dojo stop                  # end the lab and clean up
dojo tutor-context         # safe metadata to paste into an AI tutor
dojo progress reset        # archive development history and start fresh
```

The default path is ordered. A passed prerequisite unlocks the next exercise;
mastery gates can require two passes with the latest successful pass using no
hints. Experienced learners can jump ahead explicitly with
`dojo start <lab> --skip-prerequisites`.

The current dogfooded curriculum contains 12 exercises: five-stage Services
and Scheduling paths, plus independent RBAC and node troubleshooting tasks.

Environment management:

```bash
dojo env status            # machines, addresses, Kubernetes nodes
dojo env list              # every profile on this machine
dojo env stop              # shut down, keep the disks
dojo env reset             # destroy and rebuild -- the escape hatch
```

## What gets built

```
                       lima user-v2 network
                       192.168.104.0/24
       ┌───────────────┬───────────────┬───────────────┐
       │               │               │               │
   terminal           cp1           worker1         worker2
   kubectl/helm    kubeadm          kubelet         kubelet
   1 GiB           3 GiB            2.5 GiB         2.5 GiB
```

Roughly 9 GiB of guest memory. Ubuntu 24.04, containerd, Kubernetes 1.35.8,
Calico, metrics-server.

The workstation is deliberately **not** part of the cluster. That is what lets
a lab stop the kubelet on a worker, or corrupt a control-plane manifest,
without also breaking the shell you are working from.

## Repeating a lab does not mean repeating the answer

Labs can declare variants. `dojo start node-not-ready` might stop the kubelet,
or it might stop containerd — same symptom in `kubectl get nodes`, different
diagnosis. The choice comes from a seed that is printed and recorded, so a
scenario can be reproduced exactly:

```bash
dojo start node-not-ready --seed 9182731
```

## Documentation

| Document | What it covers |
|---|---|
| [ROADMAP.md](ROADMAP.md) | What exists, what is next, which decisions are settled |
| [docs/architecture.md](docs/architecture.md) | How the engine, provider and content fit together |
| [docs/authoring-labs.md](docs/authoring-labs.md) | Writing a lab: schema, faults, graders, hints |
| [docs/curriculum.md](docs/curriculum.md) | Domains, skills, modules, and how progress is scored |
| [docs/AI_TUTOR.md](docs/AI_TUTOR.md) | Using an AI assistant as a tutor, without it giving the game away |

## Development

```bash
make check        # go vet, go test, dojo content validate
make validate     # content linting on its own
```

`DOJO_CONTENT=$PWD ./bin/dojo ...` reads curriculum and environments from the
working tree instead of the copy embedded in the binary.
