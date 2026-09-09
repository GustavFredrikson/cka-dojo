# cka-dojo

CKA practice on real, disposable Kubernetes clusters. Go, no external
services; content in `curriculum/` and `environments/` is embedded in the
binary. `docs/architecture.md` explains the layers, `docs/authoring-labs.md`
the rules for exercises.

## Dogfooding must not write the study history

**This repository is also its user's study tool.** `~/.cka-dojo/progress.json`
is real study history. Running an exercise to prove it works records the same
attempt a learner's own work records, and `dojo recommend`, `dojo learn` and
`dojo readiness` cannot tell them apart — a synthetic pass demotes the
exercise in the ranking and unlocks every exercise behind it.

Before running `dojo start`, `dojo grade` or anything else that touches a lab:

```bash
make dogfood
```

The `.dojo-dev` marker it writes is gitignored and sends this checkout's
attempts to `progress-dev.json` instead. Prefer the marker over `DOJO_DEV=1`
per command: a shell in a tool call does not survive to the next one, and
forgetting the export once is enough to corrupt the history. Every dojo
command in dev mode prints a banner, so check for it.

Never reset or edit `progress.json` to clean up after dogfooding. It is the
user's own record; `dojo progress reset` is theirs to run.

The user may be mid-lab while you work. `state.json`, the lock and the cluster
are shared, so `dojo start`, `reset`, `stop` and `env` commands can destroy
work in progress — check `dojo status` first, and ask.

## Before you commit

```bash
make check      # vet, test, content validate
```
