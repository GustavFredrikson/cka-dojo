# Worked solution

First prove the application and Service path, then compare policies with Pod
labels and actual destination port:

```bash
kubectl -n dojo-netpol-contextual get pod --show-labels
kubectl -n dojo-netpol-contextual get svc,endpointslice
kubectl -n dojo-netpol-contextual get networkpolicy -o yaml
```

The hidden variants mismatch source selector, destination selector or port.
Correct only that rule and test from both client and intruder.
