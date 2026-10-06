# Join a worker to the cluster

The cluster works: `cp1` is a Ready control plane with a pod network, and your
workstation kubeconfig reaches it.

`worker1` is prepared exactly as cp1 was — containerd, the kubelet, kubeadm and
kubectl are all installed — and it is not part of the cluster.

A Deployment is stuck because of it:

```
$ kubectl -n dojo-kubeadm-join-worker get deploy,pods
NAME                    READY   UP-TO-DATE   AVAILABLE
deployment.apps/batch   0/2     2            0

NAME                     READY   STATUS    RESTARTS   AGE
pod/batch-...            0/1     Pending   0          1m
pod/batch-...            0/1     Pending   0          1m
```

Join `worker1` to the cluster, so that the `batch` Deployment becomes fully
available.

Leave `worker1` schedulable when you are done — a node that joins and stays
cordoned has not finished joining.

**Note:** this lab builds a cluster during setup, so `dojo start` and
`dojo reset` each take five to seven minutes.
