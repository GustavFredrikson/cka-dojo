# Worked solution

## Working it out

The error names the address:

```
The connection to the server k8s-api:6443 was refused
```

*Refused*, not *timed out*: something resolved the name and the connection was
actively rejected, so the host is up and nothing is listening on that port.

Establish how far the failure reaches before touching anything:

```bash
ssh cp1 'sudo kubectl --kubeconfig /etc/kubernetes/admin.conf get nodes'
```

That works. So the cluster is healthy and the problem is between your kubectl
and it. Then find out what `k8s-api` actually is:

```bash
kubectl config view --minify        # server: https://k8s-api:6443
grep k8s-api /etc/hosts             # resolves to the workstation itself
ss -ltn | grep 6443                 # nothing listening
systemctl status haproxy            # inactive (dead), masked
```

The endpoint every client uses is a load balancer running on the workstation,
and it is down.

## Fixing it

```bash
sudo systemctl unmask haproxy
sudo systemctl enable --now haproxy
kubectl get nodes
```

## The shortcut that is not a fix

```bash
# Do not do this here.
kubectl config set-cluster kubernetes --server=https://192.168.104.x:6443
```

It works instantly and it throws away the point of the cluster. Your kubectl
would then depend on one specific control plane, and the next time *that* one
went down you would be editing the file again. Grading checks the `server:` is
still `https://k8s-api:6443`.

## The lesson worth keeping

`--control-plane-endpoint` makes a cluster survive losing a control plane, and
in exchange it introduces a component that can fail on its own. A single haproxy
is a single point of failure — a production HA cluster pairs it with keepalived
and a virtual IP, or uses a cloud load balancer, precisely so this failure mode
does not exist.

Knowing that the endpoint is a separate thing that can break, and that a
`connection refused` to it says nothing about the health of the cluster behind
it, is the exam-relevant part.
