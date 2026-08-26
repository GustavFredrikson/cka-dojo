# Worked solution

```bash
kubectl -n kube-system get pods
ssh cp1
sudo ls -l /etc/kubernetes/manifests /etc/kubernetes/*scheduler*
sudo mv /etc/kubernetes/kube-scheduler.yaml.dojo-disabled /etc/kubernetes/manifests/kube-scheduler.yaml
exit
kubectl -n dojo-controlplane-scheduler get pod pending -w
```
