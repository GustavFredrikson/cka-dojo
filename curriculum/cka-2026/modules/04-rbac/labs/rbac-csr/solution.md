# Worked solution

## The idea

Kubernetes has no `User` object. An identity is whatever the authenticator
says it is, and for certificate authentication that is read straight out of
the certificate's subject:

- **CN** becomes the username.
- each **O** becomes a group.

So "create a user" means "get a certificate signed by a CA the API server
trusts, with the right subject" — and then write RBAC that names it. The two
halves are independent, which is the thing to internalise: a valid certificate
with no RoleBinding authenticates fine and can do nothing at all.

## 1. Key and CSR

```bash
openssl genrsa -out auditor.key 2048
openssl req -new -key auditor.key -out auditor.csr \
  -subj "/CN=dojo-auditor/O=dojo-auditors"
```

`-subj` order does not matter. What matters is that `CN` is exactly the name
your RoleBinding will reference.

## 2. Submit it

The CSR object carries the request base64-encoded, and `base64 -w0` keeps it on
one line — a wrapped blob is the most common reason this step fails:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: certificates.k8s.io/v1
kind: CertificateSigningRequest
metadata:
  name: dojo-auditor
spec:
  request: $(base64 -w0 auditor.csr)
  signerName: kubernetes.io/kube-apiserver-client
  usages:
    - client auth
  expirationSeconds: 86400
EOF
```

`signerName` is the field that decides what the certificate is good for:

- `kubernetes.io/kube-apiserver-client` — a client certificate for
  authenticating *to* the API server. This one.
- `kubernetes.io/kubelet-serving` — a kubelet's serving certificate.
- `kubernetes.io/kube-apiserver-client-kubelet` — what a kubelet uses to
  authenticate as `system:node:<name>`.

Pick the wrong signer and the certificate is issued and then rejected at
authentication time, which is a confusing way to lose ten minutes.

`usages: ["client auth"]` is required for this signer and must be exactly that.

## 3. Approve and extract

```bash
kubectl get csr dojo-auditor
kubectl certificate approve dojo-auditor
kubectl get csr dojo-auditor -o jsonpath='{.status.certificate}' \
  | base64 -d > auditor.crt
```

`Pending` → `Approved,Issued`. Approval is a write to a *subresource*
(`certificatesigningrequests/approval`), which is why it needs its own RBAC
permission and why `kubectl edit` is not how you do it.

`.status.certificate` is empty until the signer has run. If it stays empty
after approval, no controller is signing for that `signerName` — on a kubeadm
cluster kube-controller-manager signs the built-in ones.

## 4. Build the kubeconfig

The clean way is three `kubectl config` commands against a fresh file:

```bash
kubectl config --kubeconfig=auditor.kubeconfig set-cluster kubernetes \
  --server="$(kubectl config view --minify -o jsonpath='{.clusters[0].cluster.server}')" \
  --certificate-authority=/etc/kubernetes/pki/ca.crt --embed-certs=true

kubectl config --kubeconfig=auditor.kubeconfig set-credentials dojo-auditor \
  --client-certificate=auditor.crt --client-key=auditor.key --embed-certs=true

kubectl config --kubeconfig=auditor.kubeconfig set-context dojo-auditor \
  --cluster=kubernetes --user=dojo-auditor

kubectl config --kubeconfig=auditor.kubeconfig use-context dojo-auditor
```

The CA file is not on the workstation, so take it from the control plane:

```bash
ssh cp1 'sudo cat /etc/kubernetes/pki/ca.crt' > ca.crt
```

and point `--certificate-authority` at that. `--embed-certs=true` matters:
without it the kubeconfig holds *paths*, and it stops working the moment
anything moves.

Forgetting `use-context` is the classic miss — every command then fails with
`context was not found`, which looks like a certificate problem and is not.

## 5. RBAC

```bash
kubectl -n dojo-rbac-csr create role pod-reader \
  --verb=get,list,watch --resource=pods

kubectl -n dojo-rbac-csr create rolebinding auditor-can-read \
  --role=pod-reader --user=dojo-auditor
```

`--user=dojo-auditor` matches the certificate's CN. You could bind
`--group=dojo-auditors` instead and get the same result through the O field —
which is the better habit when more than one person will hold such a
certificate, because adding a person then means issuing a certificate rather
than editing RBAC.

A `Role` and `RoleBinding`, not their Cluster equivalents: the task says this
namespace only, and a ClusterRoleBinding would grant it everywhere.

## Checking before you grade

```bash
kubectl --kubeconfig auditor.kubeconfig auth whoami
kubectl --kubeconfig auditor.kubeconfig -n dojo-rbac-csr get pods
kubectl --kubeconfig auditor.kubeconfig -n dojo-rbac-csr auth can-i delete pods
kubectl --kubeconfig auditor.kubeconfig -n kube-system auth can-i get pods
```

`auth whoami` should report `dojo-auditor` and the group `dojo-auditors`; it
answers "who does the cluster think I am", which separates an authentication
problem from an authorisation one in one command. Then: Pods listed, `no` to
deleting, `no` in any other namespace.
