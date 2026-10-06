# Worked solution

## Working it out

There are two separate things here and only the first is obvious.

`kubeadm certs renew all` rewrites the certificate files under
`/etc/kubernetes/pki` and the embedded credentials in the kubeconfigs under
`/etc/kubernetes`. That is a file operation. It does not tell anything that is
already running to re-read them.

The control-plane components are static Pods, and kubelet restarts one only
when its *manifest* in `/etc/kubernetes/manifests` changes. Renewal does not
touch the manifests, so nothing restarts on its own.

Which certificates that actually matters for is worth getting right, because
the obvious test is misleading:

```bash
openssl x509 -in /etc/kubernetes/pki/apiserver.crt -noout -serial
openssl s_client -connect 127.0.0.1:6443 </dev/null 2>/dev/null \
  | openssl x509 -noout -serial
```

Those two agree immediately after renewal, with no restart. kube-apiserver
watches its own **serving** certificate and hot-reloads it. If you were checking
your work that way you would conclude you were finished.

What does *not* reload is everything read once at startup:

- the API server's **client** certificates — `apiserver-etcd-client` and
  `apiserver-kubelet-client`, which it presents when it calls etcd and the
  kubelets;
- the kubeconfigs `kube-controller-manager` and `kube-scheduler` authenticate
  with, `controller-manager.conf` and `scheduler.conf`.

Those components keep using the credentials they loaded at boot until they are
restarted. Nothing visibly breaks today — the old certificates are still valid,
they are simply about to stop being — which is exactly why this step gets
skipped and then bites a year later.

## Fixing it

```bash
ssh cp1
sudo kubeadm certs renew all
sudo kubeadm certs check-expiration
```

Then make kubelet restart the control plane. Moving the manifest directory out
and back restarts all four components at once:

```bash
sudo mv /etc/kubernetes/manifests /etc/kubernetes/manifests.off
sudo crictl ps --name kube-apiserver      # wait until it is gone
sudo mv /etc/kubernetes/manifests.off /etc/kubernetes/manifests
sudo crictl ps --name kube-apiserver      # wait until it is back
```

**`sudo systemctl restart kubelet` does not do this**, and it is the advice you
will most often be given. kubelet comes back, reconciles against what the
runtime is already running, finds static Pods whose manifests are unchanged, and
leaves them exactly where they are. The containers keep their original age and
their original credentials. Check with `crictl ps` rather than assuming.

The API server is unavailable for twenty seconds or so during the move, which is
worth knowing before doing it on something that matters.

## Checking before you grade

```bash
sudo kubeadm certs check-expiration
sudo crictl ps --name kube-apiserver
sudo crictl ps --name kube-scheduler
sudo kubectl --kubeconfig /etc/kubernetes/admin.conf get --raw=/readyz
```

The `AGE` column from `crictl ps` is the one that matters: those containers must
be younger than the renewal. Grading compares each container's start time
against the mtime of `/etc/kubernetes/pki/apiserver.crt`, for exactly that
reason.

One thing to be careful about: `kubeadm certs renew all` also rewrites
`/etc/kubernetes/admin.conf`. The workstation has its own copy at
`~/.kube/config`, which is *not* rewritten — but it does not need to be, because
the client certificate inside it is still valid and still signed by the same CA.
Renewal does not rotate the CA. If you ever do rotate the CA, every kubeconfig
in the cluster has to be rebuilt.

## Why this lab is repeatable even though renewal is not reversible

Grading does not compare against a fixed date. The setup step records the
current certificate *serial numbers*, and grading asserts they have changed. On
reset it records the serials you just produced, so the exercise is to renew
again — which always works, and leaves the cluster healthier each time.
