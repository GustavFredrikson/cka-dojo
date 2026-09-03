#!/usr/bin/env bash
# metrics-server, so `kubectl top` works.
#
# The cluster stays close to vanilla kubeadm, because that is the better
# teaching surface. The exceptions are components the published curriculum
# examines but that a bare kubeadm cluster cannot provide at all -- a dynamic
# provisioner, an Ingress controller and a Gateway API implementation. Those
# live in dynamic-storage.sh and ingress.sh, pinned like everything else.
set -euo pipefail
export KUBECONFIG={{.Kubeconfig}}

kubectl apply -f "https://github.com/kubernetes-sigs/metrics-server/releases/download/{{.MetricsVersion}}/components.yaml"

# kubelet serving certificates are self-signed in a kubeadm cluster.
if ! kubectl -n kube-system get deploy metrics-server \
    -o jsonpath='{.spec.template.spec.containers[0].args}' | grep -q 'kubelet-insecure-tls'; then
  kubectl -n kube-system patch deployment metrics-server --type=json \
    -p '[{"op":"add","path":"/spec/template/spec/containers/0/args/-","value":"--kubelet-insecure-tls"}]'
fi

kubectl -n kube-system rollout status deployment/metrics-server --timeout=5m
echo "metrics-server ready"
