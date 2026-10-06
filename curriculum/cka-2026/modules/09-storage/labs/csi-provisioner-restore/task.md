# Dynamic provisioning has stopped

A claim in `dojo-csi-provisioner-restore` has been `Pending` since it was
created, and the Pod that needs it will not start:

```
$ kubectl -n dojo-csi-provisioner-restore get pvc,pod
NAME                            STATUS    VOLUME   CAPACITY   STORAGECLASS
persistentvolumeclaim/reports   Pending                       local-path

NAME            READY   STATUS    RESTARTS   AGE
pod/archiver    0/1     Pending   0          1m
```

The StorageClass exists and is spelled correctly, the claim asks for 128Mi, and
there is plenty of disk on every node. Nothing else in the cluster is unhealthy.

Make the claim bind, so that `archiver` starts and can write to `/data`.

Fix whatever is supposed to provision the volume. Creating a PersistentVolume by
hand would satisfy the claim and is not the exercise — grading checks which
provisioner actually made the volume.
