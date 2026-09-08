#!/usr/bin/env bash
# etcdctl and etcdutl on the control-plane node.
#
# The real exam gives you these on the control plane, and every backup and
# restore task depends on them. There is deliberately no version pin in the
# environment profile: the client has to match the server's storage format,
# and kubeadm -- not this repo -- chooses the etcd version. So the version is
# read back out of the static Pod manifest kubeadm wrote.
#
# etcd 3.6 moved `snapshot restore` out of etcdctl and into etcdutl, so both
# binaries are installed and the labs accept either spelling where both work.
set -euo pipefail

MANIFEST=/etc/kubernetes/manifests/etcd.yaml
if [ ! -f "$MANIFEST" ]; then
  echo "no etcd static Pod manifest at $MANIFEST; is this a stacked-etcd control plane?" >&2
  exit 1
fi

# registry.k8s.io/etcd:3.6.5-0 -> 3.6.5
TAG="$(awk '/image:.*etcd/ {print $2}' "$MANIFEST" | head -1 | sed 's/.*://')"
if [ -z "$TAG" ]; then
  echo "could not read the etcd image tag from $MANIFEST" >&2
  exit 1
fi
VERSION="v${TAG%-*}"

if [ -x /usr/local/bin/etcdctl ] && [ -x /usr/local/bin/etcdutl ] &&
   etcdctl version 2>/dev/null | grep -q "etcdctl version: ${VERSION#v}"; then
  echo "etcdctl ${VERSION} already installed"
  exit 0
fi

# Preferred route: lift the binaries out of the image containerd has already
# pulled. Exact version match, and no network.
extract_from_image() {
  local ref mnt
  ref="$(ctr -n k8s.io images ls -q | grep -m1 -E "etcd:${TAG}\$")" || return 1
  [ -n "$ref" ] || return 1
  mnt="$(mktemp -d)"
  ctr -n k8s.io images mount --rw "$ref" "$mnt" >/dev/null 2>&1 || return 1
  local ok=0
  if [ -x "$mnt/usr/local/bin/etcdctl" ]; then
    install -m 0755 "$mnt/usr/local/bin/etcdctl" /usr/local/bin/etcdctl
    # etcdutl only exists from 3.5 onwards.
    [ -x "$mnt/usr/local/bin/etcdutl" ] &&
      install -m 0755 "$mnt/usr/local/bin/etcdutl" /usr/local/bin/etcdutl
    ok=1
  fi
  ctr -n k8s.io images unmount "$mnt" >/dev/null 2>&1 || true
  rmdir "$mnt" 2>/dev/null || true
  [ "$ok" = 1 ]
}

# Fallback: the upstream release tarball for the same version.
extract_from_release() {
  local arch dir
  arch="$(dpkg --print-architecture)"
  dir="etcd-${VERSION}-linux-${arch}"
  curl -fsSL -o /tmp/etcd.tgz \
    "https://github.com/etcd-io/etcd/releases/download/${VERSION}/${dir}.tar.gz" || return 1
  tar -xzf /tmp/etcd.tgz -C /tmp "${dir}/etcdctl" "${dir}/etcdutl" || return 1
  install -m 0755 "/tmp/${dir}/etcdctl" /usr/local/bin/etcdctl
  install -m 0755 "/tmp/${dir}/etcdutl" /usr/local/bin/etcdutl
  rm -rf /tmp/etcd.tgz "/tmp/${dir}"
}

if ! extract_from_image; then
  echo "could not extract etcdctl from the etcd image; falling back to the release tarball"
  if ! extract_from_release; then
    echo "could not install etcdctl ${VERSION} from either the image or the release tarball" >&2
    exit 1
  fi
fi

# The client flags are long and every lab needs them. A profile snippet keeps
# the exam-realistic invocation available without hiding it: the labs still
# ask for the full flags, this only saves retyping them at the shell.
cat > /etc/profile.d/dojo-etcd.sh <<'PROFILE'
# The kubeadm client certificate paths, for convenience only. Labs expect you
# to know where these come from: the etcd static Pod manifest.
export ETCDCTL_API=3
export ETCD_CACERT=/etc/kubernetes/pki/etcd/ca.crt
export ETCD_CLIENT_CERT=/etc/kubernetes/pki/etcd/server.crt
export ETCD_CLIENT_KEY=/etc/kubernetes/pki/etcd/server.key
PROFILE

echo "etcd tooling ready: $(etcdctl version | tr '\n' ' ')"
