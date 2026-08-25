# Curriculum model

The engine and the training content are intentionally separate. Kubernetes
versions, exam weights, skills, lessons and labs live under
`curriculum/cka-2026`; changing them does not require changing Go code.

## Exam profile

`curriculum.yaml` pins the exam-facing facts:

```yaml
id: cka-2026
certification:
  name: CKA
  curriculumRevision: "2025-02-18"
kubernetes:
  minor: "1.35"
```

The minor is not `latest`. The exam version and upstream Kubernetes are
different concepts. Environment profiles pin the exact patch used by the VMs;
the curriculum pins the minor that its tasks and lessons teach.

The five domain weights sum to 100 and mirror the published CKA weighting:

| Domain | Weight |
|---|---:|
| Troubleshooting | 30% |
| Cluster architecture, installation and configuration | 25% |
| Services and networking | 20% |
| Workloads and scheduling | 15% |
| Storage | 10% |

`dojo content validate` rejects unknown domains and weights that do not sum to
100.

## Modules are for learning order

Exam domains are scoring categories; they are not a good teaching sequence.
Modules provide that sequence:

```text
modules/05-services/
├── module.yaml
├── lesson.md
└── labs/
```

A module may touch more than one domain. A troubleshooting lab about a broken
Service belongs to both `services-networking` and `troubleshooting`, while the
module still has one clear place in the learning path.

Lessons stay concise. Their job is to give the learner a mental model and a
diagnostic workflow before practice, not to duplicate the Kubernetes docs.
The standard structure is:

```text
# Topic
## Mental model
## Objects involved
## Commands worth knowing
## Diagnostic workflow
## Common CKA failure modes
## 5-minute walkthrough
## Labs
```

## Skills are the unit of progress

Every lab declares the capabilities it exercises:

```yaml
skills:
  - services
  - selectors
  - endpointslices
```

This lets `dojo progress` answer a useful question — “Which abilities still
need work?” — rather than only reporting that a number of labs were opened.

The current mastery rule is deliberately simple:

```text
Mastered =
  at least two successful attempts
  AND the latest successful attempt used zero hints
```

Reading the solution prevents that attempt from being counted as a clean pass.
Time is displayed and best times are kept, but speed is not part of mastery
yet. Correct independent work comes first.

## Attempts and reproducibility

An attempt begins with `dojo start`, not with `dojo grade`. Grading repeatedly
while working is normal and does not inflate the attempt or pass counters.

Variant choice is derived from a stored seed. Progress therefore records the
last seed and variant, and `dojo progress --by-lab` shows the seed needed to
replay a scenario exactly:

```bash
dojo start services-no-endpoints --seed 9182731
```

Development attempts or a deliberate fresh start can be cleared without
destroying history:

```bash
dojo progress reset
```

The old `progress.json` is moved to a timestamped file in `~/.cka-dojo`; it is
never deleted outright.

## Recommendations

`dojo recommend` ranks labs with one transparent formula:

```text
score = average published domain weight × mastery gap × recency
```

An attempted-but-unpassed lab has the largest mastery gap. A new lab comes
next, followed by a passed but unmastered lab. Mastered labs remain low-weight
spaced-review candidates. Recency rises linearly for fourteen days after the
last attempt. `dojo recommend --all` prints every factor used in the ranking.

## Expanding the curriculum

Add content because study exposed a gap, then run and reset every variant by
hand. Do not generate a large catalogue before the primitives are proven.

For each new lab:

1. Add any genuinely new skill to `curriculum.yaml`.
2. Put the lab in the module that gives it the clearest learning context.
3. Declare every applicable exam domain.
4. Validate with `make check`.
5. Start the untouched lab and prove grading fails for the intended reason.
6. Solve it from the student workstation and prove grading passes.
7. Reset it and prove the fault returns.
8. Repeat for every variant.

See [authoring-labs.md](authoring-labs.md) for the complete schema and the
available fault and grader vocabulary.
