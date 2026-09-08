# Issue a client certificate for a new user

An auditor needs read-only access to Pods in namespace `dojo-rbac-csr`.
Kubernetes has no User object, so you will create the identity the way the
cluster actually understands one: a client certificate signed by the cluster
CA, plus RBAC that names its subject.

Work on the workstation as `student`, in the home directory.

1. Generate a 2048-bit RSA key `auditor.key` and a certificate signing request
   with Common Name `dojo-auditor` and Organisation `dojo-auditors`.
2. Submit it as a `CertificateSigningRequest` named `dojo-auditor`, using
   signer `kubernetes.io/kube-apiserver-client` and usage `client auth`.
3. Approve it, and save the issued certificate to `auditor.crt`.
4. Build a kubeconfig at `~/auditor.kubeconfig` that reaches this cluster and
   authenticates with that key and certificate.
5. Grant that identity `get`, `list` and `watch` on Pods **in
   `dojo-rbac-csr` only**. Nothing else, and nowhere else.

Verify it yourself before grading:

```
kubectl --kubeconfig ~/auditor.kubeconfig auth whoami
kubectl --kubeconfig ~/auditor.kubeconfig -n dojo-rbac-csr get pods
kubectl --kubeconfig ~/auditor.kubeconfig -n dojo-rbac-csr auth can-i delete pods
```
