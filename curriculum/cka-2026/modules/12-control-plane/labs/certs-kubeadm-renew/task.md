# Renew the cluster's own certificates

Nothing is broken. This is the maintenance operation you would run on a cluster
whose certificates are approaching a year old.

On `cp1`, renew every certificate kubeadm manages, and make the running control
plane actually serve the new ones.

When you are done:

- `kubeadm certs check-expiration` shows every certificate freshly issued, with
  well over 300 days left
- the API server, controller-manager and scheduler are all running on the new
  material — their containers started *after* the certificates were rewritten
- `kubectl get --raw=/readyz` returns `ok`, and your own `kubectl` from the
  workstation still works

Renewing the files is the first half, and `kubeadm` will tell you what the
second half is when it finishes. Read what it says.
