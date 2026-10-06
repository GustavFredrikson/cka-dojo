# Read the upgrade plan

This cluster is on Kubernetes 1.34 and the exam's minor is 1.35. Before
upgrading anything, find out what upgrading would involve.

**Change nothing.** This is the one exercise in this module you can repeat — the
two after it consume the environment, because a kubeadm upgrade cannot be
undone. Use this one to get the shape of the task clear first.

Answer each checkpoint with `dojo check <answer>`.

```bash
ssh cp1
kubeadm version -o short
cat /etc/apt/sources.list.d/kubernetes.list
apt-mark showhold
sudo kubeadm upgrade plan
```

`kubeadm upgrade plan` reaches out to dl.k8s.io to find the newest available
patch, so it needs the VM to have working internet — it does.
