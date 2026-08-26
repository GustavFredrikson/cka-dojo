# One workload bypasses cluster DNS

Deployment `dnscheck` in namespace `dojo-dns-policy` is running, but cannot
resolve `kubernetes.default.svc.cluster.local`. Other workloads resolve cluster
names normally.

Repair only this Deployment. Do not modify CoreDNS or node resolver files.
