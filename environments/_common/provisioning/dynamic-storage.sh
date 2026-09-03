#!/usr/bin/env bash
# The local-path provisioner, so that dynamic provisioning is practisable.
#
# A kubeadm cluster on plain VMs has no CSI driver at all, which would leave
# "implement storage classes and dynamic volume provisioning" untrainable. This
# is the smallest real provisioner that fills that hole.
set -euo pipefail
export KUBECONFIG={{.Kubeconfig}}

kubectl apply -f "https://raw.githubusercontent.com/rancher/local-path-provisioner/{{.LocalPathVersion}}/deploy/local-path-storage.yaml"

# Deliberately NOT the default class.
#
# Several storage exercises rely on a claim staying Pending. A default class
# would silently provision those claims and delete the lesson, so strip the
# annotation if upstream ever ships it set.
kubectl annotate storageclass local-path \
  storageclass.kubernetes.io/is-default-class- --overwrite >/dev/null 2>&1 || true

kubectl -n local-path-storage rollout status deployment/local-path-provisioner --timeout=5m

if kubectl get storageclass -o jsonpath='{.items[?(@.metadata.annotations.storageclass\.kubernetes\.io/is-default-class=="true")].metadata.name}' | grep -q .; then
  echo "WARNING: a default StorageClass exists; Pending-claim exercises may not behave" >&2
fi

echo "local-path provisioner ready (provisioner rancher.io/local-path, no default class)"
