# Restore scheduling on the control plane

Pod `pending` in namespace `dojo-controlplane-scheduler` remains unscheduled.
The API server and existing workloads are healthy, but a control-plane
component is absent.

Diagnose the component on `cp1`, restore its kubeadm static Pod manifest to the
correct directory, and verify that `pending` becomes scheduled. Do not recreate
the Pod.
