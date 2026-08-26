# Allow one client and deny another

Namespace `dojo-netpol-build` contains Service/Deployment `server`, client Pods
labelled `role=client`, intruder Pods labelled `role=intruder`, and default-deny
ingress.

Create NetworkPolicy `allow-client` so only `role=client` Pods may reach
`app=server` on TCP port 80. The intruder must remain denied.
