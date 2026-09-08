# Solution

## Working it out

The error message is the diagnosis. Read it before you change anything.

```bash
kubectl get nodes
```

**`connection refused`** -- you reached the host and nothing was listening on
that port. Nothing is wrong with your credentials; you are knocking on the
wrong door. Compare the `server:` field against reality.

**`x509: certificate signed by unknown authority`** -- you reached the API
server, it presented its serving certificate, and the CA in your kubeconfig
could not vouch for it. `certificate-authority-data` is wrong.

**`Error in configuration: context was not found for specified context: ...`**
-- `kubectl` never got as far as the network. `current-context` names a
context the file does not define. Note that this one is not even a server
error: it is `kubectl` refusing to start.

Three layers, three fixes. The habit worth building is the ordering: what does
the message tell you *worked* before it failed?

```bash
kubectl config view              # redacted, readable
kubectl config view --raw        # the actual base64 blobs
kubectl config get-contexts
```

You are cluster-admin on the nodes and `ssh cp1` works, so a known-good
kubeconfig for this cluster is always within reach:

```bash
ssh cp1 'sudo cat /etc/kubernetes/admin.conf'
```

Diff that against yours and the wrong field is obvious.

## Fixing it

Surgically, once you know which field it is:

```bash
kubectl config set-cluster kubernetes --server=https://<cp1-ip>:6443
kubectl config use-context kubernetes-admin@kubernetes
```

Or wholesale, which is what most people do under time pressure and is not
cheating -- kubeadm's `admin.conf` *is* the correct file:

```bash
ssh cp1 'sudo cat /etc/kubernetes/admin.conf' > ~/.kube/config
chmod 600 ~/.kube/config
```

What does not count is `export KUBECONFIG=/some/other/file`. It works for
exactly as long as that shell lives, and the next question in the exam opens
in a new one. Grading checks `~/.kube/config` itself for that reason.

## Checking before you grade

```bash
kubectl get nodes
kubectl auth can-i create pods --all-namespaces
kubectl config current-context
```

All three nodes listed, `yes`, and a context that exists. If `get nodes` works
but `can-i` says `no`, you fixed the connection and lost the identity --
you are talking to the right cluster as the wrong user.
