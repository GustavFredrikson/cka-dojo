# A workload that admission will not accept

Deployment `report` in namespace `dojo-psa` reports no available replicas, and
`kubectl -n dojo-psa get pods` shows nothing at all.

Get it running two replicas.

The namespace's security policy is deliberate and must stay exactly as strict
as it is now.
