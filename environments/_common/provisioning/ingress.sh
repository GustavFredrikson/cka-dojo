#!/usr/bin/env bash
# Ingress and Gateway API, the two north-south traffic APIs the exam examines.
#
# Both controllers are exposed through NodePort: this cluster has no cloud load
# balancer, and hostNetwork would put the two of them in a fight over :80.
set -euo pipefail
export KUBECONFIG={{.Kubeconfig}}

# --- Ingress -----------------------------------------------------------------
kubectl apply -f "https://raw.githubusercontent.com/kubernetes/ingress-nginx/{{.IngressNginxVersion}}/deploy/static/provider/baremetal/deploy.yaml"

# The admission webhook rejects Ingress objects until its certificate job has
# run. Without this wait the first exercise to create an Ingress fails with a
# connection error that has nothing to teach.
kubectl -n ingress-nginx rollout status deployment/ingress-nginx-controller --timeout=5m
kubectl -n ingress-nginx wait --for=condition=complete job/ingress-nginx-admission-create --timeout=3m >/dev/null 2>&1 || true
kubectl -n ingress-nginx wait --for=condition=complete job/ingress-nginx-admission-patch --timeout=3m >/dev/null 2>&1 || true

# --- Gateway API -------------------------------------------------------------
# CRDs first: the controller crash-loops if its API types are missing.
kubectl apply -f "https://github.com/kubernetes-sigs/gateway-api/releases/download/{{.GatewayAPIVersion}}/standard-install.yaml"
kubectl wait --for=condition=established --timeout=2m \
  crd/gatewayclasses.gateway.networking.k8s.io \
  crd/gateways.gateway.networking.k8s.io \
  crd/httproutes.gateway.networking.k8s.io

# NGF's own CRDs are a separate bundle. Its deploy manifest contains
# NginxGateway and NginxProxy *instances*, so applying it first fails with
# "no matches for kind" on both.
#
# --server-side is required, not stylistic: client-side apply records the whole
# manifest in a last-applied-configuration annotation, and the NginxProxy
# schema alone is larger than the 262144-byte annotation ceiling.
kubectl apply --server-side --force-conflicts \
  -f "https://raw.githubusercontent.com/nginx/nginx-gateway-fabric/{{.NGFVersion}}/deploy/crds.yaml"
kubectl wait --for=condition=established --timeout=2m \
  crd/nginxgateways.gateway.nginx.org \
  crd/nginxproxies.gateway.nginx.org

kubectl apply -f "https://raw.githubusercontent.com/nginx/nginx-gateway-fabric/{{.NGFVersion}}/deploy/nodeport/deploy.yaml"

# nginx-gateway is only the control plane, and it restarts a few times until
# its cert-generator Job has run. NGF v2 provisions the actual nginx data
# plane per Gateway, so there is nothing else to wait for here -- the
# NodePort Service appears when an exercise creates its first Gateway.
kubectl -n nginx-gateway wait --for=condition=complete job/nginx-gateway-cert-generator --timeout=3m >/dev/null 2>&1 || true
kubectl -n nginx-gateway rollout status deployment/nginx-gateway --timeout=5m

echo "ingress-nginx and Gateway API ready"
kubectl get ingressclass
kubectl get gatewayclass
