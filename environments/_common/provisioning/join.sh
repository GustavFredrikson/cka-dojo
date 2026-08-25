#!/usr/bin/env bash
# Join a worker using a token minted moments ago on the control plane.
set -euo pipefail

if [ -f /etc/kubernetes/kubelet.conf ]; then
  echo "{{.Node}} already joined"
  exit 0
fi

{{.JoinCommand}} \
  --node-name="{{.Node}}" \
  --cri-socket=unix:///run/containerd/containerd.sock

echo "{{.Node}} joined the cluster"
