# Worked solution

```bash
ssh cp1
sudo kubectl run dojo-static --image=busybox:1.36 --restart=Never \
  --labels='dojo.cka/static=exam' --dry-run=client -o yaml \
  --command -- sleep 3600 | sudo tee /etc/kubernetes/manifests/dojo-static.yaml
exit
kubectl get pod -l dojo.cka/static=exam -o wide
```
