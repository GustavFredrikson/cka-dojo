# Run work once, and run it on a schedule

Work in namespace `dojo-jobs-build`.

**1. A Job named `migrate`** that must succeed three times, running at most
two Pods at once, and giving up after two failed Pods.

- image `busybox:1.36`
- command: `sh -c 'echo migrating; sleep 3'`

Wait for it to complete.

**2. A CronJob named `report`** that would run every five minutes, but which
is currently **suspended** so that nothing is scheduled yet.

- schedule `*/5 * * * *`
- image `busybox:1.36`, command `sh -c 'echo reporting'`
- if a run is still going when the next is due, skip the new one
- keep only one successful Job in history
