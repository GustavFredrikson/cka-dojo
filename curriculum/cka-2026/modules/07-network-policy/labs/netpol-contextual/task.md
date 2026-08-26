# One client cannot reach a healthy Service

In namespace `dojo-netpol-contextual`, Service `server` has healthy endpoints.
Pods labelled `role=client` should reach it on port 80, while Pods labelled
`role=intruder` must remain denied. The intended client currently times out.

Restore only the required path without removing default deny.
