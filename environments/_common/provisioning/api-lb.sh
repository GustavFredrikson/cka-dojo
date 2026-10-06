#!/usr/bin/env bash
# The API server load balancer: the external LB the kubeadm HA documentation
# describes, in the simplest form that actually works here.
#
# It runs on the workstation rather than on a control plane on purpose. The
# address behind {{.ControlPlaneEndpointName}} is baked into the API server
# certificate's SANs at `kubeadm init` and cannot be moved afterwards, so it
# belongs on the one machine labs are never allowed to break -- and a load
# balancer that is also one of its own backends is a footgun that only shows up
# when that backend dies.
#
# haproxy starts happily with backends down, which is what resolves the
# chicken-and-egg: when kubeadm init runs, only the first control plane exists.
set -euo pipefail
export DEBIAN_FRONTEND=noninteractive
apt-get -y -o DPkg::Lock::Timeout=900 -o Acquire::Retries=3 install -qq haproxy

cat > /etc/haproxy/haproxy.cfg <<'CFG'
global
  log /dev/log local0
  maxconn 4000

defaults
  mode    tcp
  log     global
  option  tcplog
  timeout connect 10s
  # Long client and server timeouts: `kubectl ... -w` and the informers inside
  # every controller hold a connection open indefinitely, and a short timeout
  # here shows up as watches mysteriously dropping every few minutes.
  timeout client  4h
  timeout server  4h

frontend kube-apiserver
  bind *:{{.ControlPlaneEndpointPort}}
  default_backend kube-apiserver

backend kube-apiserver
  option tcp-check
  balance roundrobin
{{- range $name, $ip := .ControlPlaneIPs}}
  server {{$name}} {{$ip}}:6443 check check-ssl verify none inter 2s fall 3 rise 2
{{- end}}
CFG

systemctl enable --now haproxy
systemctl restart haproxy

# Prove it is listening before provisioning moves on to kubeadm init, which
# will fail confusingly if the endpoint does not answer.
for _ in $(seq 1 30); do
  if ss -ltn | grep -q ':{{.ControlPlaneEndpointPort}} '; then
    echo "load balancer listening on {{.ControlPlaneEndpoint}}"
    exit 0
  fi
  sleep 1
done
echo "haproxy is not listening on port {{.ControlPlaneEndpointPort}}" >&2
exit 1
