# Worked solution

## Working it out

`Unauthorized` is an *authentication* failure, not an authorization one. If the
Role were missing you would get a 403 naming the user; a 401 means the API
server never accepted the identity in the first place. So the Role and
RoleBinding are not worth looking at — the credential is.

A kubeconfig credential is a client certificate, and certificates have dates:

```bash
kubectl --kubeconfig ~/ops.kubeconfig config view --raw --minify \
  -o jsonpath='{.users[0].user.client-certificate-data}' \
  | base64 -d | openssl x509 -noout -subject -issuer -dates
```

`notAfter` is in the past. The subject is `CN=dojo-operator,O=dojo-operators`
and the issuer is the cluster CA, which tells you both what to reissue and who
has to sign it.

The client-side error message is deliberately unhelpful here, and that is worth
knowing for the exam. kube-apiserver asks for a client certificate but does not
require one at the TLS layer, so the handshake succeeds and the x509
authenticator rejects the certificate afterwards — you get `Unauthorized`, not
a TLS error. The real reason is on cp1:

```bash
ssh cp1 'sudo crictl logs "$(sudo crictl ps -q --name kube-apiserver)" 2>&1 | tail -40'
```

which says `verifying certificate ... failed: x509: certificate has expired or
is not yet valid`.

## Fixing it

A certificate cannot be extended; the CA has to sign a new one with the same
subject. Any of these three work.

**With kubeadm**, which is the shortest and the one to reach for under time:

```bash
ssh cp1 'sudo kubeadm kubeconfig user --client-name dojo-operator \
  --org dojo-operators --validity-period 720h' > ~/ops.kubeconfig
chmod 600 ~/ops.kubeconfig
```

**By hand with openssl**, if you want to see each step:

```bash
ssh cp1
openssl genrsa -out /tmp/ops.key 2048
openssl req -new -key /tmp/ops.key -out /tmp/ops.csr \
  -subj "/CN=dojo-operator/O=dojo-operators"
sudo openssl x509 -req -in /tmp/ops.csr -CA /etc/kubernetes/pki/ca.crt \
  -CAkey /etc/kubernetes/pki/ca.key -CAcreateserial -days 365 -out /tmp/ops.crt
```

then embed the new pair into the existing file:

```bash
kubectl --kubeconfig ~/ops.kubeconfig config set-credentials dojo-operator \
  --client-certificate=ops.crt --client-key=ops.key --embed-certs=true
```

**Through the API**, as `rbac-csr` taught: a CertificateSigningRequest with
signer `kubernetes.io/kube-apiserver-client`, approved with
`kubectl certificate approve`, then read `.status.certificate` back out.

## Checking before you grade

```bash
kubectl --kubeconfig ~/ops.kubeconfig auth whoami
kubectl --kubeconfig ~/ops.kubeconfig -n dojo-certs-expired-kubeconfig get deploy
kubectl --kubeconfig ~/ops.kubeconfig -n dojo-certs-expired-kubeconfig auth can-i delete pods
```

The first must say `dojo-operator` — not `kubernetes-admin`. Copying
`~/.kube/config` over `~/ops.kubeconfig` makes the first two commands pass and
fails the third, and grading checks all three.

One trap worth naming: if you add the renewed credential as a *second* user
entry and leave `current-context` pointing at the old one, `auth whoami` still
works (kubectl falls back through the file) but the context still selects the
expired certificate. Grading resolves the certificate through `--minify`, which
follows `current-context`, so that does not pass either.

## A note on why this lab exists

The cluster's own certificates are valid for a year and there is no supported
way to backdate them, so no lab can expire *them*. This one signs a separate
client certificate against the same CA with a `notAfter` in the past. The
diagnosis and the repair are identical to the real case; only the blast radius
differs.
