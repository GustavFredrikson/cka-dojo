# Build a control plane with kubeadm

Three machines are running. All of them have containerd, the kubelet, kubeadm
and kubectl installed at the version this environment pins, swap is off and the
sysctls are set — and none of them is part of a cluster. There is no
`~/.kube/config` on the workstation, because there is nothing to point it at.

Turn `cp1` into a working single-node control plane.

- the pod network must be planned as `10.244.0.0/16`
- the service network must be `10.96.0.0/12`
- the API server must advertise cp1's address **on the environment network**
  (`192.168.104.0/24`), not a loopback address. Be careful here: this is Ubuntu,
  so `cp1` also resolves to `127.0.1.1` locally, and kubeadm will refuse it
- when you are done, `kubectl get nodes` must work **from the workstation, as
  `student`, with no environment variables set and no sudo**

`worker1` stays out of it. You will join it in a later exercise.

**Expected, and not a mistake:** when you finish, `cp1` will report `NotReady`
and the CoreDNS Pods will sit `Pending`. There is no pod network installed yet —
that is the next exercise. Everything else should be running.

`ssh cp1` works from the workstation, and `student` has passwordless sudo on
every machine.
