# Run an agent on every node

In namespace `dojo-daemonset-build`, create a DaemonSet named `node-agent`
that runs one Pod on **every** node in the cluster, including the control
plane.

Each Pod runs:

- image `busybox:1.36`
- command `sleep 3600`
- label `app: node-agent`

When you are done, the DaemonSet must report 3 desired and 3 ready.
