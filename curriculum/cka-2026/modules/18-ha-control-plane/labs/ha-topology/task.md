# Read a highly-available control plane

This cluster has three control planes and one worker, and your kubectl does not
talk to any of them directly.

Nothing is broken and nothing you do should change that. Work out how the
high availability is actually assembled, and answer each checkpoint with
`dojo check <answer>`.

```bash
kubectl config view --minify
grep k8s-api /etc/hosts
kubectl get nodes
kubectl -n kube-system get pods -l component=kube-apiserver
kubectl -n kube-system get configmap kubeadm-config -o yaml

ssh cp1
sudo etcdctl --endpoints=https://127.0.0.1:2379 \
  --cacert=/etc/kubernetes/pki/etcd/ca.crt \
  --cert=/etc/kubernetes/pki/etcd/server.crt \
  --key=/etc/kubernetes/pki/etcd/server.key \
  member list -w table
```
