# Worked solution

## Working it out

Start by reading the failure, not the config. There are two entirely different
answers hiding behind "cannot do anything":

```bash
kubectl --kubeconfig ~/billing.kubeconfig -n dojo-certs-identity-refused get deploy
```

- **`error: You must be logged in to the server (Unauthorized)`** — a 401. The
  API server never accepted the identity. Authorization was never consulted, so
  the Role and RoleBinding are irrelevant.
- **`Error from server (Forbidden): ... is forbidden: User "..." cannot list ...`**
  — a 403. The identity *was* accepted. Read the user name in that message
  carefully; it is the single most useful string in this exercise.

Then read the credential, which is where all three answers live:

```bash
kubectl --kubeconfig ~/billing.kubeconfig config view --raw --minify \
  -o jsonpath='{.users[0].user.client-certificate-data}' \
  | base64 -d | openssl x509 -noout -subject -issuer -dates
```

and compare the issuer with the cluster's own CA:

```bash
ssh cp1 'sudo openssl x509 -in /etc/kubernetes/pki/ca.crt -noout -subject'
```

### Telling the three apart

| What you see | What the certificate shows | What is wrong |
|---|---|---|
| 401 | subject right, issuer right, `notAfter` in the past | it expired |
| 401 | subject right, `notAfter` fine, issuer is **not** the cluster CA | signed by an authority the cluster does not trust |
| 403 naming `dojo-b1lling` | everything valid, subject CN is subtly misspelt | a valid credential for the wrong person |

The API server log distinguishes the two 401s directly:

```bash
ssh cp1 'sudo crictl logs $(sudo crictl ps -q --name kube-apiserver) 2>&1 | tail -40'
```

`certificate has expired or is not yet valid` for the first;
`certificate signed by unknown authority` for the second. The 403 case appears
nowhere in that log, because authentication succeeded — which is a usable signal
in itself.

## Fixing it

All three have the same repair: the cluster CA signs a new certificate with the
correct subject, and it replaces the old one in the same file.

```bash
ssh cp1 'sudo kubeadm kubeconfig user --client-name dojo-billing \
  --org dojo-billing --validity-period 720h' > ~/billing.kubeconfig
chmod 600 ~/billing.kubeconfig
```

By hand, if you prefer — note `-CA` must be the cluster's CA, which is the
whole point of the `foreign-ca` variant:

```bash
ssh cp1
openssl genrsa -out /tmp/billing.key 2048
openssl req -new -key /tmp/billing.key -out /tmp/billing.csr \
  -subj "/CN=dojo-billing/O=dojo-billing"
sudo openssl x509 -req -in /tmp/billing.csr \
  -CA /etc/kubernetes/pki/ca.crt -CAkey /etc/kubernetes/pki/ca.key \
  -CAcreateserial -days 365 -out /tmp/billing.crt
```

## What is not a fix

The `subject-typo` variant is the one that tempts you into the wrong repair. The
credential authenticates as `dojo-b1lling`, so binding the Role to *that* name
makes the account work — and is wrong. You have just granted a permanent
identity to a typo. The account is `dojo-billing`; reissue the certificate.

Grading asserts the RoleBinding still names `dojo-billing` and that no
ClusterRoleBinding mentions the account at all, so both of the tempting
shortcuts fail.

## Checking before you grade

```bash
kubectl --kubeconfig ~/billing.kubeconfig auth whoami
kubectl --kubeconfig ~/billing.kubeconfig -n dojo-certs-identity-refused get deploy
kubectl --kubeconfig ~/billing.kubeconfig -n dojo-certs-identity-refused auth can-i delete deployments
kubectl -n dojo-certs-identity-refused get rolebinding billing -o yaml
```

`whoami` must say `dojo-billing`, the third must say `no`, and the RoleBinding
must be exactly as you found it.
