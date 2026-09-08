# Worked solution

## The Job

`kubectl create job` gets you the skeleton, but not the three fields the task
is actually testing -- there are no flags for them:

```bash
kubectl -n dojo-jobs-build create job migrate --image=busybox:1.36 \
  --dry-run=client -o yaml -- sh -c 'echo migrating; sleep 3' > job.yaml
```

Add them by hand:

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: migrate
  namespace: dojo-jobs-build
spec:
  completions: 3
  parallelism: 2
  backoffLimit: 2
  template:
    spec:
      restartPolicy: OnFailure
      containers:
        - name: migrate
          image: busybox:1.36
          command: ["sh", "-c", "echo migrating; sleep 3"]
```

The three numbers answer three different questions:

- `completions: 3` -- how many Pods must exit 0 before the Job is done.
- `parallelism: 2` -- how many may be in flight at once. Two run, then the
  third starts as one finishes.
- `backoffLimit: 2` -- how many *failed Pods* are tolerated before the Job
  gives up. Not restarts, and not per-Pod: it is a total, and the delay
  between retries doubles each time.

`restartPolicy` is required, and must be `Never` or `OnFailure`. The API
rejects `Always`, which is the default a bare Pod template gets -- so a
hand-written Job that omits it fails validation rather than misbehaving.

## The CronJob

```bash
kubectl -n dojo-jobs-build create cronjob report --image=busybox:1.36 \
  --schedule='*/5 * * * *' --dry-run=client -o yaml \
  -- sh -c 'echo reporting' > cj.yaml
```

`create cronjob` does take `--schedule`, and also `--suspend`. The rest goes
in by hand:

```yaml
spec:
  schedule: "*/5 * * * *"
  suspend: true
  concurrencyPolicy: Forbid
  successfulJobsHistoryLimit: 1
  jobTemplate:
    spec:
      template:
        spec:
          restartPolicy: OnFailure
          containers:
            - name: report
              image: busybox:1.36
              command: ["sh", "-c", "echo reporting"]
```

Note the nesting: `CronJob.spec.jobTemplate.spec.template.spec` is four
`spec`s deep, because a CronJob contains a Job which contains a Pod. Getting
lost in it is the single most common way to lose this question; `kubectl
explain cronjob.spec.jobTemplate.spec.template.spec` is faster than counting
indentation.

"Skip the new one" is `concurrencyPolicy: Forbid`. The alternatives are
`Allow` (the default -- let them overlap) and `Replace` (kill the running one).

`suspend: true` stops new Jobs being created while leaving the object and its
history alone. It is the answer to "stop this from running" that does not
throw away the schedule, and `kubectl patch cronjob report -p '{"spec":{"suspend":true}}'`
is how you would do it to a live one.

## Checking before you grade

```bash
kubectl -n dojo-jobs-build get job migrate
kubectl -n dojo-jobs-build get cronjob report
```

The Job should read `COMPLETIONS 3/3`. The CronJob should read
`SUSPEND true` and `ACTIVE 0`, with `LAST SCHEDULE` empty -- if it has a last
schedule, it ran before you suspended it, which is harmless here but means
`suspend` went in second.
