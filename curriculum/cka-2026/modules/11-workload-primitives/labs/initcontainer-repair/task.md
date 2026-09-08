# A rollout that never starts

Deployment `web` in namespace `dojo-initcontainer` has never become
available. Its Pods are not crashing and not being rejected -- they simply do
not start.

Find out what is holding them, and satisfy it.

Leave the Deployment's Pod template alone: the init container is there
deliberately and must still be there when you are done.
