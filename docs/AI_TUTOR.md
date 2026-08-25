# Using an AI tutor

The dojo is complete without an AI service. Tasks, hints, solutions, grading
and progress all run locally, with no API key and no training repository
mounted into the student VM.

An AI assistant can still be useful as a tutor if it protects the diagnostic
work instead of immediately revealing the repair. Paste the prompt below into
the assistant you already use.

## Tutor prompt

```text
You are my CKA tutor.

For an active dojo lab:
- do not reveal the complete solution unless I explicitly ask for it;
- first ask what I inspected and what evidence I found;
- help me distinguish observations from assumptions;
- explain Kubernetes concepts when I ask, but keep the active fault hidden;
- give progressively stronger hints: area, object, diagnostic approach, then
  useful commands;
- prefer terminal workflows and state-based verification;
- accept any technically correct solution, not only the command you expected;
- after I finish, review correctness, safety, and efficiency;
- call out commands that worked accidentally or granted permissions too broadly;
- keep recommendations compatible with Kubernetes 1.35 and a kubeadm cluster.

Context I will provide:
- curriculum and Kubernetes version;
- active lab, mode, skills and target time;
- hints already used;
- the commands I ran and their output.

Never ask me to paste solution.md, the lab fault definition, or the grader
definition while the attempt is active.
```

## What to share

Safe context includes:

- the task text shown by `dojo task`;
- command output you obtained as the student;
- the active lab id, skills, mode, elapsed time and hint count;
- your current diagnosis and what evidence would disprove it.

Do not share `lab.yaml`, `solution.md`, grader definitions or fault
definitions during an attempt. Those describe the answer and turn tutoring
into answer retrieval.

`dojo tutor-context` prints only that safe metadata in one pasteable block:

```bash
dojo tutor-context
```

Its formatter uses an explicit allow-list and never receives the active
scenario's fault, grader or solution. It also omits the selected variant and
seed. Use `dojo status` and `dojo task` when you additionally want to share the
learner-visible task text.

## A useful tutoring loop

```text
Learner:  Here is the task. I see worker1 is NotReady.
Tutor:    What does the node condition say, and what have you checked on the
          node itself?
Learner:  The status is Unknown. kubelet is active.
Tutor:    Good. If kubelet is running but cannot report useful status, what
          dependency does it need to manage containers? Check that dependency
          and then read the kubelet's recent log entries.
```

The tutor narrows the search without naming the stopped service. That preserves
the part of the exercise the CKA actually tests.
