# Repair cross-namespace service discovery

Deployment `frontend` in namespace `dojo-dns-frontend` cannot start because it
cannot resolve the healthy Service `api` in namespace `dojo-dns-backend`.

Correct the frontend command to use the Service's fully qualified cluster DNS
name. Do not create another Service or change CoreDNS.
