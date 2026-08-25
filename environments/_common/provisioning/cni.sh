#!/usr/bin/env bash
# Calico, installed through the Tigera operator so the pod CIDR and address
# autodetection are declared in an API object rather than sed'ed into a
# DaemonSet manifest.
set -euo pipefail
export KUBECONFIG={{.Kubeconfig}}

# Server-side apply: the Calico CRDs are far past the annotation size limit
# that client-side apply imposes.
kubectl apply --server-side --force-conflicts \
  -f "https://raw.githubusercontent.com/projectcalico/calico/{{.CalicoVersion}}/manifests/operator-crds.yaml"
kubectl apply --server-side --force-conflicts \
  -f "https://raw.githubusercontent.com/projectcalico/calico/{{.CalicoVersion}}/manifests/tigera-operator.yaml"

kubectl -n tigera-operator rollout status deploy/tigera-operator --timeout=5m

# VXLAN everywhere with BGP off: the nodes share a subnet, but the provider
# network will not route pod addresses for us, so always encapsulate.
kubectl apply -f - <<'CALICO'
apiVersion: operator.tigera.io/v1
kind: Installation
metadata:
  name: default
spec:
  calicoNetwork:
    bgp: Disabled
    nodeAddressAutodetectionV4:
      cidrs:
      - "{{.NodeSubnet}}"
    ipPools:
    - name: default-ipv4-ippool
      blockSize: 26
      cidr: "{{.PodSubnet}}"
      encapsulation: VXLAN
      natOutgoing: Enabled
      nodeSelector: all()
---
apiVersion: operator.tigera.io/v1
kind: APIServer
metadata:
  name: default
spec: {}
CALICO

echo "waiting for calico-node"
for i in $(seq 1 60); do
  if kubectl -n calico-system get daemonset/calico-node >/dev/null 2>&1; then
    break
  fi
  sleep 10
done
kubectl -n calico-system rollout status daemonset/calico-node --timeout=10m
kubectl -n kube-system rollout status deployment/coredns --timeout=5m
echo "calico ready"
