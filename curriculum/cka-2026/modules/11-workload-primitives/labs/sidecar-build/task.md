# Ship a log file that never reaches stdout

Deployment `app` in namespace `dojo-sidecar` writes its log to
`/var/log/app/app.log` inside an `emptyDir`, and prints nothing to stdout — so
`kubectl logs` on it returns nothing useful.

Add a **sidecar container** to the Deployment that makes those lines available
through `kubectl logs`:

- named `log-shipper`
- image `busybox:1.36`
- reads the same log file the app writes, and streams it to stdout
- must keep running for as long as the Pod does, and must be started before
  the app container

Do not change the `app` container.

When you are done, `kubectl -n dojo-sidecar logs deployment/app -c log-shipper`
must show the app's entries.
