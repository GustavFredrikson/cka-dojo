# The API server will not come up

`kubectl` no longer works from the workstation. Every command ends the same
way:

```
The connection to the server <control-plane>:6443 was refused - did you specify the right host or port?
```

The nodes are running and nobody has touched your kubeconfig.

Find out why the control plane is not serving and put it back into service, so
that `kubectl` works again and all three nodes report `Ready`.

The cluster's existing data must survive. If you find yourself about to
restore a backup, you have the wrong diagnosis.
