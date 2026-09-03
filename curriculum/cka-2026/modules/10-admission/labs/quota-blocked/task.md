# A Deployment with no Pods at all

Deployment `worker` in namespace `dojo-quota-blocked` reports `0/2` replicas
available. `kubectl get pods -n dojo-quota-blocked` lists nothing whatsoever —
not Pending, not CrashLoopBackOff, nothing.

The image is valid and all three nodes are Ready with capacity to spare.

Get both replicas running. The namespace's existing governance objects must
still be in force when you are finished.
