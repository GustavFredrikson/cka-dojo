# Worked solution

## The listing

```bash
ssh cp1
sudo kubeadm certs check-expiration
```

Two blocks come out. The first is the leaf certificates and the embedded
client certificates in the kubeconfig files -- `admin.conf`, `apiserver`,
`apiserver-etcd-client`, `apiserver-kubelet-client`, `controller-manager.conf`,
`etcd-healthcheck-client`, `etcd-peer`, `etcd-server`, `front-proxy-client`,
`scheduler.conf`, and on recent versions `super-admin.conf`. All are issued
for **365 days**.

The second block is the certificate authorities: `ca`, `etcd-ca` and
`front-proxy-ca`, valid for **10 years** (3650 days). This is why an
unattended cluster dies after a year rather than after a decade: the CAs
outlive everything they signed.

The `CERTIFICATE AUTHORITY` column in the first block tells you which CA
signed each leaf, which is the map you need when something reports
`unknown authority`.

## Where they live

```bash
sudo ls -l /etc/kubernetes/pki /etc/kubernetes/pki/etcd
sudo ls -l /etc/kubernetes/*.conf
```

`/etc/kubernetes/pki` holds the files on disk. The `*.conf` kubeconfigs carry
their client certificate *inside* the file as base64, which is why they appear
in the expiry listing but not in the directory.

## Reading a certificate by hand

`kubeadm certs check-expiration` does not print subjects, and the subject is
what decides authorisation. For a file:

```bash
sudo openssl x509 -in /etc/kubernetes/pki/apiserver.crt -noout -subject -dates -ext subjectAltName
```

For one embedded in a kubeconfig, extract and decode it first:

```bash
sudo grep client-certificate-data /etc/kubernetes/admin.conf \
  | awk '{print $2}' | base64 -d | openssl x509 -noout -subject -dates
```

```
subject=O=kubeadm:cluster-admins, CN=kubernetes-admin
```

Both halves matter, and Kubernetes reads them differently:

- **CN** becomes the username -- `kubernetes-admin`.
- **O** becomes a group -- `kubeadm:cluster-admins`.

The privilege comes from the group, not the name. kubeadm creates a
ClusterRoleBinding granting `cluster-admin` to `kubeadm:cluster-admins`:

```bash
kubectl get clusterrolebinding kubeadm:cluster-admins -o yaml
```

Before Kubernetes 1.29, kubeadm put `admin.conf` in `system:masters` instead,
which is hard-coded in the API server to bypass authorisation entirely and
cannot be revoked with RBAC. That is what `super-admin.conf` is now for, and
why it exists as a separate file you are expected to leave alone.

## The one kubeadm does not manage

```bash
sudo ls -l /var/lib/kubelet/pki
```

`kubelet-client-current.pem` is a symlink the kubelet rotates itself, through
the CSR API, well before expiry. It is deliberately absent from
`kubeadm certs check-expiration` -- nothing needs to renew it by hand. The
serving certificate beside it, `kubelet.crt`, is self-signed, which is why
metrics-server on this cluster runs with `--kubelet-insecure-tls`.

## Renewal, for when you do need it

```bash
sudo kubeadm certs renew all          # then restart the control-plane Pods
sudo kubeadm certs renew apiserver    # one at a time
```

`renew all` rewrites the files but does not restart anything: the static Pods
keep serving the certificates they loaded at startup. Move the manifests out
of `/etc/kubernetes/manifests` and back, or reboot the node.

Renewal also does **not** update `/etc/kubernetes/admin.conf` on its own in
older versions, and never updates a copy you made at `~/.kube/config`. After
renewing, refresh your own kubeconfig from `admin.conf` again.
