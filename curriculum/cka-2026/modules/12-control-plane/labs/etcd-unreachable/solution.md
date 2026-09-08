# Solution

## Working it out

The message tells you the API server is not listening on `cp1:6443`. It says
nothing about why, and no `kubectl` command will, because every one of them
goes through the thing that is down. So the first move is off the workstation
and onto the node:

```bash
ssh cp1
```

Static Pods are started by kubelet, not by the scheduler, so they keep running
-- and keep crashing -- with no API server at all. That means the container
runtime still knows everything:

```bash
sudo crictl ps -a | grep -E 'apiserver|etcd'
```

etcd is `Running`. kube-apiserver is `Exited`, and has been restarted several
times. That ordering is the whole diagnosis: the component that is failing is
*not* the component that is broken. Read its logs:

```bash
sudo crictl logs "$(sudo crictl ps -a -q --name kube-apiserver | head -1)" 2>&1 | tail -30
```

```
W0907 09:44:28.619063  1 logging.go:55] [core] [Channel #2 SubChannel #6]grpc:
addrConn.createTransport failed to connect to {Addr: "127.0.0.1:2379", ...}.
Err: connection error: desc = "transport: Error while dialing: dial tcp
127.0.0.1:2379: connect: connection refused"
```

It is a wall of gRPC noise and it repeats forever, which is why it is worth
knowing what to look for rather than reading it: an address, and
`connection refused`.

The API server cannot reach etcd. But etcd is running -- so check where it is
actually listening:

```bash
sudo ss -ltnp | grep -E '2379|2380|2381'
```

Port 2379 is absent; something is bound to 12379 instead. Now compare the two
manifests:

```bash
sudo grep -n 'urls' /etc/kubernetes/manifests/etcd.yaml
sudo grep -n 'etcd-servers' /etc/kubernetes/manifests/kube-apiserver.yaml
```

etcd advertises and listens on `12379`; the API server dials `2379`.

Two things are worth noticing about why etcd looked healthy. Its liveness and
readiness probes use `--listen-metrics-urls`, which is on `2381` and was never
touched -- a component can pass its own probes and still be unreachable by
everything that needs it. And peer traffic is on `2380`, so a single-member
cluster has no complaint either.

## Fixing it

Restore the client port in `/etc/kubernetes/manifests/etcd.yaml`:

```bash
sudo sed -i 's/:12379/:2379/g' /etc/kubernetes/manifests/etcd.yaml
```

kubelet notices the file changing and restarts the Pod within seconds.

Editing `--etcd-servers` in the API server manifest to point at `12379`
instead would also produce a working cluster, and grading accepts it. It is
the worse answer: every other client of etcd -- your own `etcdctl`
invocations, a future kubeadm upgrade, anything reading the kubeadm defaults
-- still assumes 2379. Put the odd one back rather than teaching everything
else about it.

Nothing here calls for a restore. The data was never touched; only the door
was moved.

## Checking before you grade

```bash
sudo crictl ps | grep -E 'apiserver|etcd'
kubectl get --raw=/readyz
kubectl get nodes
kubectl -n dojo-etcd-unreachable get configmap intact
```

Allow thirty to sixty seconds for the API server to restart and for both
workers to be marked `Ready` again -- they were never actually down, but their
leases went stale while the API server was.
