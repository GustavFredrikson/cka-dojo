# A Deployment that never gets a node

Deployment `reporting` in namespace `dojo-observability-oversized-requests`
has never had a single replica available. The image is correct and the cluster
has three healthy nodes with idle capacity.

Measurements from staging show each replica needs roughly `100m` of CPU and
`64Mi` of memory in steady state.

Get both replicas running. The container must keep an explicit CPU request
that reflects what the workload actually needs.
