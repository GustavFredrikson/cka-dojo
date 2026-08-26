# Worked solution

Create a PVC with `storageClassName: dojo-static`, request `512Mi`, and
`accessModes: [ReadWriteOnce]`. Then create Pod `writer` with:

```yaml
volumes:
  - name: data
    persistentVolumeClaim: {claimName: data}
containers:
  - name: writer
    image: busybox:1.36
    command: [sh, -c, "sleep 3600"]
    volumeMounts: [{name: data, mountPath: /data}]
```
