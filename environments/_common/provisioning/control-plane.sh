#!/usr/bin/env bash
# kubeadm init. Idempotent: an already-initialised control plane is left alone.
set -euo pipefail

if [ -f /etc/kubernetes/admin.conf ]; then
  echo "control plane already initialised"
  exit 0
fi

kubeadm init \
  --kubernetes-version="v{{.K8sVersion}}" \
{{- if .ControlPlaneEndpoint}}
  --control-plane-endpoint="{{.ControlPlaneEndpoint}}" \
  --upload-certs \
{{- end}}
  --apiserver-advertise-address="{{.NodeIP}}" \
  --pod-network-cidr="{{.PodSubnet}}" \
  --service-cidr="{{.ServiceSubnet}}" \
  --node-name="{{.Node}}" \
  --cri-socket=unix:///run/containerd/containerd.sock

mkdir -p /root/.kube
install -m 0600 /etc/kubernetes/admin.conf /root/.kube/config
echo "control plane initialised on {{.NodeIP}}"
