#!/usr/bin/env bash
# metrics-server, so `kubectl top` works. Nothing else is installed by
# default: a mostly vanilla kubeadm cluster is the better teaching surface,
# and per-module components get installed by the modules that need them.
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
